using System.Runtime.InteropServices;
using System.Text;
using System.Text.Json;
using Realms;

namespace LazerBridge;

/// <summary>
/// Exposes an osu!lazer library to danser as an osu!stable-like directory tree (Songs/, Skins/, Replays/)
/// made of symlinks to lazer's hashed files. Nothing is copied and lazer's data is opened read-only.
/// </summary>
public static class Program
{
    public static int Main(string[] args)
    {
        string? lazerDir = null;
        string? outDir = null;
        bool json = false;

        for (int i = 0; i < args.Length; i++)
        {
            switch (args[i])
            {
                case "--lazer": lazerDir = args[++i]; break;
                case "--out": outDir = args[++i]; break;
                case "--json": json = true; break;
                case "-h":
                case "--help":
                    PrintUsage();
                    return 0;
                default:
                    Console.Error.WriteLine($"Unknown argument: {args[i]}");
                    PrintUsage();
                    return 2;
            }
        }

        lazerDir ??= DefaultLazerDirectory();

        if (outDir == null)
        {
            Console.Error.WriteLine("--out is required");
            PrintUsage();
            return 2;
        }

        try
        {
            var stats = Sync(lazerDir, Path.GetFullPath(outDir));

            Console.WriteLine(json
                ? JsonSerializer.Serialize(stats)
                : $"Synced {stats.Beatmapsets} beatmap sets, {stats.Skins} skins, {stats.Replays} replays " +
                  $"({stats.Created} links created, {stats.Removed} removed, {stats.Missing} files missing)");

            return 0;
        }
        catch (Exception e)
        {
            Console.Error.WriteLine($"lazer-bridge: {e.Message}");
            return 1;
        }
    }

    static void PrintUsage()
    {
        Console.Error.WriteLine("Usage: lazer-bridge --out <dir> [--lazer <osu!lazer data dir>] [--json]");
        Console.Error.WriteLine($"Default lazer dir: {DefaultLazerDirectory()}");
    }

    public static string DefaultLazerDirectory()
    {
        string home = Environment.GetFolderPath(Environment.SpecialFolder.UserProfile);

        if (RuntimeInformation.IsOSPlatform(OSPlatform.Windows))
            return Path.Combine(Environment.GetFolderPath(Environment.SpecialFolder.ApplicationData), "osu");

        if (RuntimeInformation.IsOSPlatform(OSPlatform.OSX))
            return Path.Combine(home, "Library", "Application Support", "osu");

        return Path.Combine(home, ".local", "share", "osu");
    }

    public record Stats(int Beatmapsets, int Skins, int Replays, int Created, int Removed, int Missing);

    static Stats Sync(string lazerDir, string outDir)
    {
        string realmPath = Path.Combine(lazerDir, "client.realm");
        if (!File.Exists(realmPath))
            throw new FileNotFoundException($"osu!lazer database not found: {realmPath}");

        string filesDir = Path.Combine(lazerDir, "files");

        // Dynamic mode reads the schema from the file itself, so lazer schema bumps don't break us
        var config = new RealmConfiguration(realmPath)
        {
            IsDynamic = true,
            IsReadOnly = true,
        };

        using var realm = Realm.GetInstance(config);

        var links = new Dictionary<string, string>(StringComparer.Ordinal);
        int missing = 0;

        string HashPath(string hash) => Path.Combine(filesDir, hash[..1], hash[..2], hash);

        void AddFiles(string dir, IEnumerable<IRealmObjectBase> files)
        {
            foreach (var usage in files)
            {
                string? name = usage.DynamicApi.Get<string?>("Filename");
                var file = usage.DynamicApi.Get<IRealmObjectBase?>("File");
                string? hash = file?.DynamicApi.Get<string>("Hash");

                if (string.IsNullOrWhiteSpace(name) || string.IsNullOrEmpty(hash))
                    continue;

                string target = HashPath(hash);
                if (!File.Exists(target))
                {
                    missing++;
                    continue;
                }

                string rel = SafeRelativePath(name);
                if (rel.Length == 0)
                    continue;

                links[Path.Combine(dir, rel)] = target;
            }
        }

        // ---- beatmaps
        int sets = 0;
        var usedNames = new HashSet<string>(StringComparer.OrdinalIgnoreCase);

        foreach (var set in realm.DynamicApi.All("BeatmapSet"))
        {
            if (set.DynamicApi.Get<bool>("DeletePending"))
                continue;

            var beatmaps = set.DynamicApi.GetList<IRealmObjectBase>("Beatmaps");
            if (beatmaps.Count == 0)
                continue;

            var metadata = beatmaps[0].DynamicApi.Get<IRealmObjectBase?>("Metadata");
            string artist = metadata?.DynamicApi.Get<string?>("Artist") ?? "";
            string title = metadata?.DynamicApi.Get<string?>("Title") ?? "";
            int onlineId = set.DynamicApi.Get<int>("OnlineID");
            string id = set.DynamicApi.Get<Guid>("ID").ToString("N")[..8];

            string prefix = onlineId > 0 ? onlineId.ToString() : "L" + id;
            string folder = Truncate(SanitizeName($"{prefix} {artist} - {title}"), 150);
            if (!usedNames.Add(folder))
            {
                folder = Truncate(SanitizeName($"{prefix} {artist} - {title}"), 140) + $" [{id}]";
                usedNames.Add(folder);
            }

            AddFiles(Path.Combine(outDir, "Songs", folder), set.DynamicApi.GetList<IRealmObjectBase>("Files"));
            sets++;
        }

        // ---- skins (protected ones are lazer's built-in skins without files)
        int skins = 0;
        usedNames.Clear();

        foreach (var skin in realm.DynamicApi.All("Skin"))
        {
            if (skin.DynamicApi.Get<bool>("DeletePending") || skin.DynamicApi.Get<bool>("Protected"))
                continue;

            string name = SanitizeName(skin.DynamicApi.Get<string?>("Name") ?? "");
            if (name.Length == 0)
                name = "Skin";

            if (!usedNames.Add(name))
            {
                name += $" [{skin.DynamicApi.Get<Guid>("ID").ToString("N")[..8]}]";
                usedNames.Add(name);
            }

            AddFiles(Path.Combine(outDir, "Skins", name), skin.DynamicApi.GetList<IRealmObjectBase>("Files"));
            skins++;
        }

        // ---- replays
        int replays = 0;

        foreach (var score in realm.DynamicApi.All("Score"))
        {
            if (score.DynamicApi.Get<bool>("DeletePending"))
                continue;

            var files = score.DynamicApi.GetList<IRealmObjectBase>("Files");
            var replayFile = files.FirstOrDefault()?.DynamicApi.Get<IRealmObjectBase?>("File");
            string? hash = replayFile?.DynamicApi.Get<string>("Hash");

            if (string.IsNullOrEmpty(hash))
                continue;

            string target = HashPath(hash);
            if (!File.Exists(target))
            {
                missing++;
                continue;
            }

            string user = score.DynamicApi.Get<IRealmObjectBase?>("User")?.DynamicApi.Get<string?>("Username") ?? "Unknown";
            var beatmap = score.DynamicApi.Get<IRealmObjectBase?>("BeatmapInfo");
            var metadata = beatmap?.DynamicApi.Get<IRealmObjectBase?>("Metadata");
            string artist = metadata?.DynamicApi.Get<string?>("Artist") ?? "";
            string title = metadata?.DynamicApi.Get<string?>("Title") ?? "";
            string diff = beatmap?.DynamicApi.Get<string?>("DifficultyName") ?? "";
            var date = score.DynamicApi.Get<DateTimeOffset>("Date");
            string id = score.DynamicApi.Get<Guid>("ID").ToString("N")[..8];

            string mapName = metadata != null
                ? $"{artist} - {title} [{diff}]"
                : $"unknown beatmap {score.DynamicApi.Get<string?>("BeatmapHash")?[..Math.Min(8, score.DynamicApi.Get<string?>("BeatmapHash")?.Length ?? 0)]}";

            string name = Truncate(SanitizeName($"{user} - {mapName}"), 180) +
                          $" ({date.LocalDateTime:yyyy-MM-dd HH-mm-ss}) {id}.osr";

            links[Path.Combine(outDir, "Replays", name)] = target;
            replays++;
        }

        var (created, removed) = Apply(outDir, links);

        return new Stats(sets, skins, replays, created, removed, missing);
    }

    /// <summary>
    /// Makes the symlink tree under outDir match the desired links. Only symlinks and empty directories are ever removed.
    /// </summary>
    static (int created, int removed) Apply(string outDir, Dictionary<string, string> links)
    {
        int created = 0, removed = 0;

        foreach (string root in new[] { "Songs", "Skins", "Replays" })
        {
            string rootDir = Path.Combine(outDir, root);
            Directory.CreateDirectory(rootDir);

            foreach (string path in Directory.EnumerateFileSystemEntries(rootDir, "*", SearchOption.AllDirectories).ToList())
            {
                var info = new FileInfo(path);
                if (info.LinkTarget == null)
                    continue;

                if (!links.TryGetValue(path, out string? target) || target != info.LinkTarget)
                {
                    File.Delete(path);
                    removed++;
                }
            }
        }

        foreach (var (path, target) in links)
        {
            var info = new FileInfo(path);
            if (info.LinkTarget == target)
                continue;

            if (info.Exists || info.LinkTarget != null)
                continue; // a real file the user put there, don't touch it

            Directory.CreateDirectory(Path.GetDirectoryName(path)!);
            File.CreateSymbolicLink(path, target);
            created++;
        }

        foreach (string root in new[] { "Songs", "Skins", "Replays" })
            RemoveEmptyDirs(Path.Combine(outDir, root), false);

        return (created, removed);
    }

    static void RemoveEmptyDirs(string dir, bool removeSelf)
    {
        foreach (string sub in Directory.EnumerateDirectories(dir))
        {
            if (new DirectoryInfo(sub).LinkTarget == null)
                RemoveEmptyDirs(sub, true);
        }

        if (removeSelf && !Directory.EnumerateFileSystemEntries(dir).Any())
            Directory.Delete(dir);
    }

    static readonly char[] invalidChars = "<>:\"/\\|?*".ToCharArray();

    static string SanitizeName(string name)
    {
        var sb = new StringBuilder(name.Length);

        foreach (char c in name)
            sb.Append(c < 32 || invalidChars.Contains(c) ? '_' : c);

        return sb.ToString().Trim().TrimEnd('.');
    }

    /// <summary>
    /// Beatmap filenames may contain subdirectories (storyboards), keep them but never allow escaping the folder.
    /// </summary>
    static string SafeRelativePath(string name)
    {
        var parts = name.Replace('\\', '/')
                        .Split('/', StringSplitOptions.RemoveEmptyEntries)
                        .Where(p => p != "." && p != "..")
                        .Select(SanitizeName)
                        .Where(p => p.Length > 0);

        return Path.Combine(parts.ToArray());
    }

    static string Truncate(string s, int max) => s.Length <= max ? s : s[..max].TrimEnd();
}

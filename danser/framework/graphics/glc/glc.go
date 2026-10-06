// Package glc provides OpenGL 3.3 core replacements for the GL 4.5 direct state access
// and storage functions danser originally relied on. macOS caps at OpenGL 4.1 core with no
// DSA, buffer storage, clear texture, copy image or base instance support, so every function
// here emulates its DSA counterpart with bind-to-edit while preserving bindings tracked in
// the history package.
package glc

import (
	"log"
	"unsafe"

	"github.com/go-gl/gl/v3.3-core/gl"

	"github.com/wieku/danser-go/framework/graphics/history"
)

// scratchUnit is a texture unit reserved for editing textures so regular bindings stay intact.
var scratchUnit uint32 = 31

var textureTargets = make(map[uint32]uint32)

var scratchFBO uint32

// Init must be called right after gl.Init.
func Init() {
	var maxUnits int32
	gl.GetIntegerv(gl.MAX_COMBINED_TEXTURE_IMAGE_UNITS, &maxUnits)

	scratchUnit = uint32(maxUnits - 1)

	gl.GenFramebuffers(1, &scratchFBO)
}

func restoreFramebuffer() {
	gl.BindFramebuffer(gl.FRAMEBUFFER, history.GetCurrent(gl.FRAMEBUFFER_BINDING))
}

// ---------------------------------------------------------------- buffers

func CreateBuffers(n int32, buffers *uint32) {
	gl.GenBuffers(n, buffers)
}

func NamedBufferData(buffer uint32, size int, data unsafe.Pointer, usage uint32) {
	gl.BindBuffer(gl.COPY_WRITE_BUFFER, buffer)
	gl.BufferData(gl.COPY_WRITE_BUFFER, size, data, usage)
	gl.BindBuffer(gl.COPY_WRITE_BUFFER, 0)
}

func NamedBufferSubData(buffer uint32, offset int, size int, data unsafe.Pointer) {
	gl.BindBuffer(gl.COPY_WRITE_BUFFER, buffer)
	gl.BufferSubData(gl.COPY_WRITE_BUFFER, offset, size, data)
	gl.BindBuffer(gl.COPY_WRITE_BUFFER, 0)
}

func ptr(data unsafe.Pointer) any {
	if data == nil {
		return gl.Ptr(nil)
	}

	return data
}

// ---------------------------------------------------------------- textures

func CreateTextures(target uint32, n int32, textures *uint32) {
	gl.GenTextures(n, textures)

	ids := unsafeSlice(textures, n)
	for _, id := range ids {
		textureTargets[id] = target
		bindScratch(id)
	}
}

func DeleteTextures(n int32, textures *uint32) {
	for _, id := range unsafeSlice(textures, n) {
		delete(textureTargets, id)
	}

	gl.DeleteTextures(n, textures)
}

func targetOf(texture uint32) uint32 {
	if t, ok := textureTargets[texture]; ok {
		return t
	}

	return gl.TEXTURE_2D_ARRAY
}

func bindScratch(texture uint32) uint32 {
	target := targetOf(texture)

	gl.ActiveTexture(gl.TEXTURE0 + scratchUnit)
	gl.BindTexture(target, texture)

	return target
}

func BindTextureUnit(unit uint32, texture uint32) {
	gl.ActiveTexture(gl.TEXTURE0 + unit)
	gl.BindTexture(targetOf(texture), texture)
}

func TextureParameteri(texture uint32, pname uint32, param int32) {
	target := bindScratch(texture)
	gl.TexParameteri(target, pname, param)
}

// TextureStorage3D emulates immutable storage with mutable TexImage3D levels.
func TextureStorage3D(texture uint32, levels int32, internalFormat uint32, width, height, depth int32, format, xtype uint32) {
	target := bindScratch(texture)

	for level := int32(0); level < levels; level++ {
		w := max(1, width>>level)
		h := max(1, height>>level)

		gl.TexImage3D(target, level, int32(internalFormat), w, h, depth, 0, format, xtype, gl.Ptr(nil))
	}
}

func TextureSubImage3D(texture uint32, level, xoffset, yoffset, zoffset, width, height, depth int32, format, xtype uint32, pixels unsafe.Pointer) {
	target := bindScratch(texture)
	gl.TexSubImage3D(target, level, xoffset, yoffset, zoffset, width, height, depth, format, xtype, pixels)
}

func GenerateTextureMipmap(texture uint32) {
	target := bindScratch(texture)
	gl.GenerateMipmap(target)
}

// ClearTexLayers clears level 0 of every layer of a 2D array texture through a scratch framebuffer.
func ClearTexLayers(texture uint32, layers int32, depth bool, color [4]float32) {
	scissor := gl.IsEnabled(gl.SCISSOR_TEST)
	gl.Disable(gl.SCISSOR_TEST)

	var mask [4]bool
	gl.GetBooleanv(gl.COLOR_WRITEMASK, &mask[0])
	gl.ColorMask(true, true, true, true)

	var depthMask bool
	gl.GetBooleanv(gl.DEPTH_WRITEMASK, &depthMask)
	gl.DepthMask(true)

	gl.BindFramebuffer(gl.FRAMEBUFFER, scratchFBO)

	for layer := int32(0); layer < layers; layer++ {
		if depth {
			gl.FramebufferTextureLayer(gl.FRAMEBUFFER, gl.DEPTH_ATTACHMENT, texture, 0, layer)
			gl.ClearBufferfv(gl.DEPTH, 0, &color[0])
		} else {
			gl.FramebufferTextureLayer(gl.FRAMEBUFFER, gl.COLOR_ATTACHMENT0, texture, 0, layer)
			gl.ClearBufferfv(gl.COLOR, 0, &color[0])
		}
	}

	if depth {
		gl.FramebufferTextureLayer(gl.FRAMEBUFFER, gl.DEPTH_ATTACHMENT, 0, 0, 0)
	} else {
		gl.FramebufferTextureLayer(gl.FRAMEBUFFER, gl.COLOR_ATTACHMENT0, 0, 0, 0)
	}

	restoreFramebuffer()

	gl.ColorMask(mask[0], mask[1], mask[2], mask[3])
	gl.DepthMask(depthMask)

	if scissor {
		gl.Enable(gl.SCISSOR_TEST)
	}
}

// CopyTexLayers copies layers [0, layers) of the given mip level between two 2D array textures.
func CopyTexLayers(src, dst uint32, level, width, height, layers int32) {
	scissor := gl.IsEnabled(gl.SCISSOR_TEST)
	gl.Disable(gl.SCISSOR_TEST)

	target := bindScratch(dst)

	gl.BindFramebuffer(gl.READ_FRAMEBUFFER, scratchFBO)

	for layer := int32(0); layer < layers; layer++ {
		gl.FramebufferTextureLayer(gl.READ_FRAMEBUFFER, gl.COLOR_ATTACHMENT0, src, level, layer)
		gl.ReadBuffer(gl.COLOR_ATTACHMENT0)
		gl.CopyTexSubImage3D(target, level, 0, 0, layer, 0, 0, width, height)
	}

	gl.FramebufferTextureLayer(gl.READ_FRAMEBUFFER, gl.COLOR_ATTACHMENT0, 0, 0, 0)

	restoreFramebuffer()

	if scissor {
		gl.Enable(gl.SCISSOR_TEST)
	}
}

// ReadTexLayer reads level 0 of a texture layer into the currently bound PIXEL_PACK_BUFFER (or memory).
func ReadTexLayer(texture uint32, layer, width, height int32, format, xtype uint32, pixels unsafe.Pointer) {
	gl.BindFramebuffer(gl.READ_FRAMEBUFFER, scratchFBO)
	gl.FramebufferTextureLayer(gl.READ_FRAMEBUFFER, gl.COLOR_ATTACHMENT0, texture, 0, layer)
	gl.ReadBuffer(gl.COLOR_ATTACHMENT0)

	gl.ReadPixels(0, 0, width, height, format, xtype, pixels)

	gl.FramebufferTextureLayer(gl.READ_FRAMEBUFFER, gl.COLOR_ATTACHMENT0, 0, 0, 0)

	restoreFramebuffer()
}

// ---------------------------------------------------------------- framebuffers

func CreateFramebuffers(n int32, framebuffers *uint32) {
	gl.GenFramebuffers(n, framebuffers)
}

func withFramebuffer(fb uint32, f func()) {
	gl.BindFramebuffer(gl.FRAMEBUFFER, fb)
	f()

	if status := gl.CheckFramebufferStatus(gl.FRAMEBUFFER); status != gl.FRAMEBUFFER_COMPLETE {
		log.Printf("glc: framebuffer %d is incomplete: 0x%X", fb, status)
	}

	restoreFramebuffer()
}

func NamedFramebufferTextureLayer(fb, attachment, texture uint32, level, layer int32) {
	withFramebuffer(fb, func() {
		gl.FramebufferTextureLayer(gl.FRAMEBUFFER, attachment, texture, level, layer)
	})
}

func NamedFramebufferRenderbuffer(fb, attachment, rbTarget, rb uint32) {
	withFramebuffer(fb, func() {
		gl.FramebufferRenderbuffer(gl.FRAMEBUFFER, attachment, rbTarget, rb)
	})
}

func NamedFramebufferDrawBuffers(fb uint32, n int32, bufs *uint32) {
	withFramebuffer(fb, func() {
		gl.DrawBuffers(n, bufs)
	})
}

func NamedFramebufferDrawBuffer(fb, buf uint32) {
	withFramebuffer(fb, func() {
		gl.DrawBuffer(buf)
	})
}

func NamedFramebufferReadBuffer(fb, buf uint32) {
	withFramebuffer(fb, func() {
		gl.ReadBuffer(buf)
	})
}

func ClearNamedFramebufferfv(fb, buffer uint32, drawBuffer int32, value *float32) {
	withFramebuffer(fb, func() {
		gl.ClearBufferfv(buffer, drawBuffer, value)
	})
}

func BlitNamedFramebuffer(src, dst uint32, srcX0, srcY0, srcX1, srcY1, dstX0, dstY0, dstX1, dstY1 int32, mask, filter uint32) {
	gl.BindFramebuffer(gl.READ_FRAMEBUFFER, src)
	gl.BindFramebuffer(gl.DRAW_FRAMEBUFFER, dst)
	gl.BlitFramebuffer(srcX0, srcY0, srcX1, srcY1, dstX0, dstY0, dstX1, dstY1, mask, filter)
	restoreFramebuffer()
}

// ---------------------------------------------------------------- renderbuffers

func CreateRenderbuffers(n int32, rbs *uint32) {
	gl.GenRenderbuffers(n, rbs)
}

func NamedRenderbufferStorage(rb, internalFormat uint32, width, height int32) {
	gl.BindRenderbuffer(gl.RENDERBUFFER, rb)
	gl.RenderbufferStorage(gl.RENDERBUFFER, internalFormat, width, height)
	gl.BindRenderbuffer(gl.RENDERBUFFER, 0)
}

func NamedRenderbufferStorageMultisample(rb uint32, samples int32, internalFormat uint32, width, height int32) {
	gl.BindRenderbuffer(gl.RENDERBUFFER, rb)
	gl.RenderbufferStorageMultisample(gl.RENDERBUFFER, samples, internalFormat, width, height)
	gl.BindRenderbuffer(gl.RENDERBUFFER, 0)
}

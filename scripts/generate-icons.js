/**
 * Generates static PNG icons for the Next.js app.
 * Uses only Node.js built-in modules (zlib, fs) — no external deps.
 * Run from repo root: node scripts/generate-icons.js
 */
'use strict';
const zlib = require('zlib');
const fs = require('fs');
const path = require('path');

// CRC32 table (PNG spec requirement for each chunk)
const CRC_TABLE = (() => {
  const t = new Uint32Array(256);
  for (let i = 0; i < 256; i++) {
    let c = i;
    for (let j = 0; j < 8; j++) c = (c & 1) ? 0xEDB88320 ^ (c >>> 1) : c >>> 1;
    t[i] = c;
  }
  return t;
})();

function crc32(buf) {
  let crc = 0xFFFFFFFF;
  for (let i = 0; i < buf.length; i++) crc = CRC_TABLE[(crc ^ buf[i]) & 0xFF] ^ (crc >>> 8);
  return (crc ^ 0xFFFFFFFF) >>> 0;
}

function pngChunk(type, data) {
  const typeBuf = Buffer.from(type, 'ascii');
  const len = Buffer.alloc(4);
  len.writeUInt32BE(data.length);
  const crcVal = Buffer.alloc(4);
  crcVal.writeUInt32BE(crc32(Buffer.concat([typeBuf, data])));
  return Buffer.concat([len, typeBuf, data, crcVal]);
}

/**
 * Build a valid RGB PNG of the given size.
 * pixelFn(x, y, w, h) => [r, g, b]  (all 0-255)
 */
function makePNG(width, height, pixelFn) {
  // Signature
  const sig = Buffer.from([137, 80, 78, 71, 13, 10, 26, 10]);

  // IHDR
  const ihdr = Buffer.alloc(13);
  ihdr.writeUInt32BE(width, 0);
  ihdr.writeUInt32BE(height, 4);
  ihdr[8] = 8; // 8-bit depth
  ihdr[9] = 2; // RGB (no alpha)
  // compression=0, filter=0, interlace=0 already zero

  // Raw scanlines: 1 filter byte + width*3 RGB bytes per row
  const stride = 1 + width * 3;
  const raw = Buffer.alloc(height * stride);
  for (let y = 0; y < height; y++) {
    raw[y * stride] = 0; // filter = None
    for (let x = 0; x < width; x++) {
      const [r, g, b] = pixelFn(x, y, width, height);
      const off = y * stride + 1 + x * 3;
      raw[off] = r;
      raw[off + 1] = g;
      raw[off + 2] = b;
    }
  }

  return Buffer.concat([
    sig,
    pngChunk('IHDR', ihdr),
    pngChunk('IDAT', zlib.deflateSync(raw, { level: 6 })),
    pngChunk('IEND', Buffer.alloc(0)),
  ]);
}

// Diagonal gradient: slate-950 #020617 (2,6,23) → slate-800 #1e293b (30,41,59)
function gradientPixel(x, y, w, h) {
  const t = (x + y) / (w + h - 2);
  return [
    Math.round(2 + 28 * t),
    Math.round(6 + 35 * t),
    Math.round(23 + 36 * t),
  ];
}

const outDir = path.join(__dirname, '..', 'apps', 'web', 'app');

const icon512 = makePNG(512, 512, gradientPixel);
fs.writeFileSync(path.join(outDir, 'icon.png'), icon512);
console.log(`icon.png       ${icon512.length} bytes`);

const icon180 = makePNG(180, 180, gradientPixel);
fs.writeFileSync(path.join(outDir, 'apple-icon.png'), icon180);
console.log(`apple-icon.png ${icon180.length} bytes`);

console.log('Done — static icons written to apps/web/app/');

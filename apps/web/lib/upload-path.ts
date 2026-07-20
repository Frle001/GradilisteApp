export interface UploadEntry {
  file: File
  relativePath: string
}

const MAX_FOLDER_DEPTH = 20
const MAX_PATH_LEN = 1024

const SYSTEM_FILES = new Set([
  '.ds_store',
  'thumbs.db',
  'desktop.ini',
  '._.ds_store',
])

export function isSystemFile(name: string): boolean {
  return SYSTEM_FILES.has(name.toLowerCase())
}

export function normalizeRelativePath(raw: string): string | null {
  let p = raw.replace(/\\/g, '/')

  if (p.includes('\x00')) return null
  if (p.startsWith('/') || /^[a-zA-Z]:/.test(p)) return null
  if (p.length > MAX_PATH_LEN) return null

  const segments = p.split('/')
  if (segments.length - 1 > MAX_FOLDER_DEPTH) return null

  for (const seg of segments) {
    if (seg === '' || seg === '.' || seg === '..') return null
  }

  return segments.join('/')
}

/**
 * Reads a FileSystemDirectoryEntry recursively and appends UploadEntry items.
 */
async function readDirectory(
  entry: FileSystemDirectoryEntry,
  prefix: string,
  out: UploadEntry[],
): Promise<void> {
  const reader = entry.createReader()
  const readBatch = (): Promise<FileSystemEntry[]> =>
    new Promise((resolve, reject) => reader.readEntries(resolve, reject))

  let batch: FileSystemEntry[]
  do {
    batch = await readBatch()
    for (const child of batch) {
      const childPath = prefix ? `${prefix}/${child.name}` : child.name
      if (child.isFile) {
        await readFileEntry(child as FileSystemFileEntry, childPath, out)
      } else if (child.isDirectory) {
        await readDirectory(child as FileSystemDirectoryEntry, childPath, out)
      }
    }
  } while (batch.length > 0)
}

async function readFileEntry(
  entry: FileSystemFileEntry,
  relativePath: string,
  out: UploadEntry[],
): Promise<void> {
  const file = await new Promise<File>((resolve, reject) =>
    entry.file(resolve, reject),
  )
  if (isSystemFile(file.name)) return
  const norm = normalizeRelativePath(relativePath)
  if (norm) out.push({ file, relativePath: norm })
}

/**
 * Converts a DataTransfer or FileList into a flat list of UploadEntry, traversing
 * directories when the DataTransfer API provides FileSystemEntry access.
 */
export async function getFilesFromDataTransfer(
  dt: DataTransfer,
): Promise<UploadEntry[]> {
  const out: UploadEntry[] = []

  if (dt.items && dt.items.length > 0) {
    const entries: FileSystemEntry[] = []
    for (let i = 0; i < dt.items.length; i++) {
      const entry = dt.items[i].webkitGetAsEntry?.()
      if (entry) entries.push(entry)
    }
    if (entries.length > 0) {
      for (const entry of entries) {
        if (entry.isFile) {
          await readFileEntry(entry as FileSystemFileEntry, entry.name, out)
        } else if (entry.isDirectory) {
          await readDirectory(
            entry as FileSystemDirectoryEntry,
            entry.name,
            out,
          )
        }
      }
      return out
    }
  }

  // Fallback: plain file list (no directory traversal).
  for (let i = 0; i < dt.files.length; i++) {
    const file = dt.files[i]
    if (isSystemFile(file.name)) continue
    const norm = normalizeRelativePath(file.name)
    if (norm) out.push({ file, relativePath: norm })
  }
  return out
}

/**
 * Converts a FileList (from <input type="file">) into UploadEntry list.
 * Preserves relative paths when the input had webkitdirectory set.
 */
export function getFilesFromFileList(fileList: FileList): UploadEntry[] {
  const out: UploadEntry[] = []
  for (let i = 0; i < fileList.length; i++) {
    const file = fileList[i]
    if (isSystemFile(file.name)) continue
    // webkitRelativePath is set when the input has webkitdirectory attribute.
    const rawPath =
      (file as File & { webkitRelativePath?: string }).webkitRelativePath ||
      file.name
    const norm = normalizeRelativePath(rawPath)
    if (norm) out.push({ file, relativePath: norm })
  }
  return out
}

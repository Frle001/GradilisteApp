import { openDB, type DBSchema, type IDBPDatabase } from 'idb'
import type { OutboxEntry } from './types'

interface GradilisteDB extends DBSchema {
  outbox: {
    key: string
    value: OutboxEntry
    indexes: {
      'by-status': string
      'by-type': string
      'by-parent': string
    }
  }
  /** Key-value store for form drafts. Keys are arbitrary strings. */
  drafts: {
    key: string
    value: unknown
  }
  /** Raw File objects queued for photo upload. Keys are UUIDs. */
  blobs: {
    key: string
    value: File
  }
}

const DB_NAME = 'gradiliste'
const DB_VERSION = 1

let _dbPromise: Promise<IDBPDatabase<GradilisteDB>> | null = null

export function getDB(): Promise<IDBPDatabase<GradilisteDB>> {
  if (typeof window === 'undefined') {
    return Promise.reject(new Error('IndexedDB is not available in this environment'))
  }
  if (!_dbPromise) {
    _dbPromise = openDB<GradilisteDB>(DB_NAME, DB_VERSION, {
      upgrade(db) {
        const outbox = db.createObjectStore('outbox', { keyPath: 'id' })
        outbox.createIndex('by-status', 'status')
        outbox.createIndex('by-type', 'type')
        outbox.createIndex('by-parent', 'parentId')

        db.createObjectStore('drafts')
        db.createObjectStore('blobs')
      },
    })
  }
  return _dbPromise
}

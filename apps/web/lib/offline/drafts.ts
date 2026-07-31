import { getDB } from './db'

export async function getDraft<T>(key: string): Promise<T | undefined> {
  const db = await getDB()
  return db.get('drafts', key) as Promise<T | undefined>
}

export async function setDraft<T>(key: string, value: T): Promise<void> {
  const db = await getDB()
  await db.put('drafts', value as unknown, key)
}

export async function clearDraft(key: string): Promise<void> {
  const db = await getDB()
  await db.delete('drafts', key)
}

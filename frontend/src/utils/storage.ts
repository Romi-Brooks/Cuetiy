import type { Message } from '../types/api'

const DB_NAME = 'CuetiyCache'
const DB_VERSION = 1
const STORE_NAME = 'kv'

function openDB(): Promise<IDBDatabase> {
  return new Promise((resolve, reject) => {
    const request = indexedDB.open(DB_NAME, DB_VERSION)
    request.onupgradeneeded = () => {
      const db = request.result
      if (!db.objectStoreNames.contains(STORE_NAME)) {
        db.createObjectStore(STORE_NAME)
      }
    }
    request.onsuccess = () => resolve(request.result)
    request.onerror = () => reject(request.error)
  })
}

async function get<T>(key: string): Promise<T | undefined> {
  const db = await openDB()
  return new Promise((resolve, reject) => {
    const tx = db.transaction(STORE_NAME, 'readonly')
    const store = tx.objectStore(STORE_NAME)
    const req = store.get(key)
    req.onsuccess = () => {
      resolve(req.result as T | undefined)
    }
    req.onerror = () => reject(req.error)
    tx.oncomplete = () => db.close()
  })
}

async function set(key: string, value: unknown): Promise<void> {
  const db = await openDB()
  return new Promise((resolve, reject) => {
    const tx = db.transaction(STORE_NAME, 'readwrite')
    const store = tx.objectStore(STORE_NAME)
    store.put(value, key)
    tx.oncomplete = () => {
      db.close()
      resolve()
    }
    tx.onerror = () => reject(tx.error)
  })
}

async function del(key: string): Promise<void> {
  const db = await openDB()
  return new Promise((resolve, reject) => {
    const tx = db.transaction(STORE_NAME, 'readwrite')
    const store = tx.objectStore(STORE_NAME)
    store.delete(key)
    tx.oncomplete = () => {
      db.close()
      resolve()
    }
    tx.onerror = () => reject(tx.error)
  })
}

const MESSAGES_PREFIX = 'chat:messages:'
const CONVERSATIONS_KEY = 'chat:conversations'

export async function getLocalMessages(convId: number): Promise<Message[]> {
  try {
    return (await get(`${MESSAGES_PREFIX}${convId}`)) || []
  } catch {
    return []
  }
}

export async function saveLocalMessages(convId: number, messages: Message[]) {
  try {
    await set(`${MESSAGES_PREFIX}${convId}`, messages)
  } catch (e) {
    console.error('IndexedDB save error:', e)
  }
}

export async function appendLocalMessage(convId: number, msg: Message) {
  try {
    const msgs = await getLocalMessages(convId)
    msgs.push(msg)
    await saveLocalMessages(convId, msgs)
  } catch (e) {
    console.error('IndexedDB append error:', e)
  }
}

export async function clearLocalMessages(convId: number) {
  try {
    await del(`${MESSAGES_PREFIX}${convId}`)
  } catch (e) {
    console.error('IndexedDB clear error:', e)
  }
}

export async function getLocalConversations() {
  try {
    return (await get(CONVERSATIONS_KEY)) || null
  } catch {
    return null
  }
}

export async function saveLocalConversations(data: unknown) {
  try {
    await set(CONVERSATIONS_KEY, data)
  } catch (e) {
    console.error('IndexedDB save conversations error:', e)
  }
}

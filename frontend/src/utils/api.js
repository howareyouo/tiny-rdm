/**
 * HTTP API adapter layer - replaces Wails RPC bindings for web mode.
 * All functions match the original Wails-generated function signatures.
 */

const API_BASE = '/api'

function unauthorized() {
    window.dispatchEvent(new Event('rdm:unauthorized'))
    return { success: false, msg: 'unauthorized' }
}

async function call(method, path, payload) {
    let url = `${API_BASE}${path}`
    const init = { method, credentials: 'same-origin' }
    if (method === 'GET' || method === 'DELETE') {
        const query = payload ? new URLSearchParams(payload).toString() : ''
        if (query) {
            url += `?${query}`
        }
    } else {
        init.headers = { 'Content-Type': 'application/json' }
        init.body = JSON.stringify(payload ?? {})
    }
    const resp = await fetch(url, init)
    if (resp.status === 401) {
        return unauthorized()
    }
    return resp.json()
}

// ==================== Connection Service ====================

export function ListConnection() {
    return call('GET', '/connection/list')
}

export function GetConnection(name) {
    return call('GET', '/connection/get', { name })
}

export function SaveConnection(name, param) {
    return call('POST', '/connection/save', { name, param })
}

export function SaveSortedConnection(conns) {
    return call('POST', '/connection/save-sorted', { conns })
}

export function TestConnection(param) {
    return call('POST', '/connection/test', param)
}

export function DeleteConnection(name) {
    return call('DELETE', '/connection/delete', { name })
}

export function CreateGroup(name) {
    return call('POST', '/connection/group/create', { name })
}

export function RenameGroup(name, newName) {
    return call('POST', '/connection/group/rename', { name, newName })
}

export function DeleteGroup(name, includeConn) {
    return call('DELETE', '/connection/group/delete', { name, includeConn })
}

export function SaveLastDB(name, db) {
    return call('POST', '/connection/save-last-db', { name, db })
}

export function SaveRefreshInterval(name, interval) {
    return call('POST', '/connection/save-refresh-interval', { name, interval })
}

export async function ExportConnections() {
    // Web mode: trigger browser download of connections zip
    try {
        const resp = await fetch(`${API_BASE}/connection/export-download`, {
            credentials: 'same-origin',
        })
        if (resp.status === 401) {
            return unauthorized()
        }
        if (!resp.ok) {
            const err = await resp.json().catch(() => ({}))
            return { success: false, msg: err.msg || 'export failed' }
        }
        const blob = await resp.blob()
        const url = URL.createObjectURL(blob)
        const a = document.createElement('a')
        const disposition = resp.headers.get('Content-Disposition') || ''
        const match = disposition.match(/filename=(.+)/)
        a.download = match ? match[1] : 'connections.zip'
        a.href = url
        a.click()
        URL.revokeObjectURL(url)
        return { success: true, data: { path: '' } }
    } catch {
        return { success: false, msg: 'export failed' }
    }
}

export async function ImportConnections() {
    // Web mode: open file picker, upload zip to backend
    return new Promise((resolve) => {
        const input = document.createElement('input')
        input.type = 'file'
        input.accept = '.zip'
        input.onchange = async () => {
            if (input.files && input.files[0]) {
                const formData = new FormData()
                formData.append('file', input.files[0])
                try {
                    const resp = await fetch(`${API_BASE}/connection/import-upload`, {
                        method: 'POST',
                        credentials: 'same-origin',
                        body: formData,
                    })
                    if (resp.status === 401) {
                        resolve(unauthorized())
                        return
                    }
                    resolve(await resp.json())
                } catch {
                    resolve({ success: false, msg: 'import failed' })
                }
            } else {
                resolve({ success: false, msg: '' })
            }
        }
        // User cancelled file picker
        input.addEventListener('cancel', () => resolve({ success: false, msg: '' }))
        input.click()
    })
}

export function ParseConnectURL(url) {
    return call('POST', '/connection/parse-url', { url })
}

export function ListSentinelMasters(param) {
    return call('POST', '/connection/list-sentinel-masters', param)
}

// ==================== Browser Service ====================

export function OpenConnection(name) {
    return call('POST', '/browser/open-connection', { name })
}

export function CloseConnection(name) {
    return call('POST', '/browser/close-connection', { name })
}

export function OpenDatabase(server, db) {
    return call('POST', '/browser/open-database', { server, db })
}

export function ServerInfo(name) {
    return call('POST', '/browser/server-info', { name })
}

export function LoadNextKeys(server, db, match, keyType, exactMatch) {
    return call('POST', '/browser/load-next-keys', { server, db, match, keyType, exactMatch })
}

export function LoadNextAllKeys(server, db, match, keyType, exactMatch) {
    return call('POST', '/browser/load-next-all-keys', { server, db, match, keyType, exactMatch })
}

export function LoadAllKeys(server, db, match, keyType, exactMatch) {
    return call('POST', '/browser/load-all-keys', { server, db, match, keyType, exactMatch })
}

export function GetKeyType(param) {
    return call('POST', '/browser/get-key-type', param)
}

export function GetKeySummary(param) {
    return call('POST', '/browser/get-key-summary', param)
}

export function GetKeyDetail(param) {
    return call('POST', '/browser/get-key-detail', param)
}

export function ConvertValue(value, decode, format) {
    return call('POST', '/browser/convert-value', { value, decode, format })
}

export function SetKeyValue(param) {
    return call('POST', '/browser/set-key-value', param)
}

export function GetHashValue(param) {
    return call('POST', '/browser/get-hash-value', param)
}

export function SetHashValue(param) {
    return call('POST', '/browser/set-hash-value', param)
}

export function AddHashField(server, db, key, action, fieldItems) {
    return call('POST', '/browser/add-hash-field', { server, db, key, action, fieldItems })
}

export function AddListItem(server, db, key, action, items) {
    return call('POST', '/browser/add-list-item', { server, db, key, action, items })
}

export function SetListItem(param) {
    return call('POST', '/browser/set-list-item', param)
}

export function SetSetItem(server, db, key, remove, members) {
    return call('POST', '/browser/set-set-item', { server, db, key, remove, members })
}

export function UpdateSetItem(param) {
    return call('POST', '/browser/update-set-item', param)
}

export function UpdateZSetValue(param) {
    return call('POST', '/browser/update-zset-value', param)
}

export function AddZSetValue(server, db, key, action, valueScore) {
    return call('POST', '/browser/add-zset-value', { server, db, key, action, valueScore })
}

export function AddStreamValue(server, db, key, id, fieldItems) {
    return call('POST', '/browser/add-stream-value', { server, db, key, id, fieldItems })
}

export function RemoveStreamValues(server, db, key, ids) {
    return call('POST', '/browser/remove-stream-values', { server, db, key, ids })
}

export function SetKeyTTL(server, db, key, ttl) {
    return call('POST', '/browser/set-key-ttl', { server, db, key, ttl })
}

export function BatchSetTTL(server, db, keys, ttl, serialNo) {
    return call('POST', '/browser/batch-set-ttl', { server, db, keys, ttl, serialNo })
}

export function DeleteKey(server, db, key, async) {
    return call('POST', '/browser/delete-key', { server, db, key, async })
}

export function DeleteKeys(server, db, keys, serialNo) {
    return call('POST', '/browser/delete-keys', { server, db, keys, serialNo })
}

export function DeleteKeysByPattern(server, db, pattern) {
    return call('POST', '/browser/delete-keys-by-pattern', { server, db, pattern })
}

export function RenameKey(server, db, key, newKey) {
    return call('POST', '/browser/rename-key', { server, db, key, newKey })
}

export function ExportKey(server, db, keys, path, includeExpire) {
    return call('POST', '/browser/export-key', { server, db, keys, path, includeExpire })
}

export function ImportCSV(server, db, path, conflict, ttl) {
    return call('POST', '/browser/import-csv', { server, db, path, conflict, ttl })
}

export function FlushDB(server, db, async) {
    return call('POST', '/browser/flush-db', { server, db, async })
}

export function GetSlowLogs(server, num) {
    return call('POST', '/browser/get-slow-logs', { server, num })
}

export function GetClientList(server) {
    return call('POST', '/browser/get-client-list', { server })
}

export function GetCmdHistory(pageNo, pageSize) {
    return call('POST', '/browser/get-cmd-history', { pageNo, pageSize })
}

export function CleanCmdHistory() {
    return call('POST', '/browser/clean-cmd-history')
}

// ==================== CLI Service ====================

export function StartCli(server, db) {
    return call('POST', '/cli/start', { server, db })
}

export function CloseCli(server) {
    return call('POST', '/cli/close', { server })
}

// ==================== Monitor Service ====================

export function StartMonitor(server) {
    return call('POST', '/monitor/start', { server })
}

export function StopMonitor(server) {
    return call('POST', '/monitor/stop', { server })
}

export function ExportLog(logs) {
    return call('POST', '/monitor/export-log', { logs })
}

// ==================== Pubsub Service ====================

export function Publish(server, channel, payload) {
    return call('POST', '/pubsub/publish', { server, channel, payload })
}

export function StartSubscribe(server) {
    return call('POST', '/pubsub/subscribe', { server })
}

export function StopSubscribe(server) {
    return call('POST', '/pubsub/unsubscribe', { server })
}

// ==================== Preferences Service ====================

export function GetPreferences() {
    return call('GET', '/preferences/get')
}

export function SetPreferences(pf) {
    return call('POST', '/preferences/set', pf)
}

export function UpdatePreferences(value) {
    return call('POST', '/preferences/update', value)
}

export function RestorePreferences() {
    return call('POST', '/preferences/restore')
}

// Common fonts to probe when Local Font Access API is unavailable
const CANDIDATE_FONTS = [
    // Sans-serif
    'Arial', 'Helvetica', 'Helvetica Neue', 'Verdana', 'Geneva', 'Tahoma',
    'Trebuchet MS', 'Lucida Grande', 'Lucida Sans Unicode', 'Segoe UI',
    'Roboto', 'Noto Sans', 'Open Sans', 'Lato', 'Source Sans Pro',
    // Serif
    'Times New Roman', 'Georgia', 'Palatino', 'Book Antiqua', 'Cambria',
    'Noto Serif',
    // Monospace
    'Courier New', 'Consolas', 'Monaco', 'Menlo', 'DejaVu Sans Mono',
    'Fira Code', 'JetBrains Mono', 'Source Code Pro', 'Cascadia Code',
    // CJK
    'Microsoft YaHei', 'PingFang SC', 'PingFang TC', 'Hiragino Sans GB',
    'Noto Sans SC', 'Noto Sans TC', 'Noto Sans JP', 'Noto Sans KR',
    'Source Han Sans SC', 'Source Han Sans TC', 'WenQuanYi Micro Hei',
    'Yu Gothic', 'Meiryo', 'Malgun Gothic',
]

async function queryBrowserFonts() {
    await document.fonts.ready
    return CANDIDATE_FONTS.filter((f) => document.fonts.check(`16px "${f}"`)).map((name) => ({ name, path: '' }))
}

export async function GetFontList() {
    try {
        const fonts = await queryBrowserFonts()
        return { success: true, data: { fonts } }
    } catch (_) {
        return { success: true, data: { fonts: [] } }
    }
}

export function GetBuildInDecoder() {
    return call('GET', '/preferences/buildin-decoder')
}

export function GetAppVersion() {
    return call('GET', '/preferences/version')
}

export function CheckForUpdate() {
    return call('GET', '/preferences/check-update')
}

// ==================== System Service ====================

// Alias used in App.vue
export function Info() {
    return call('GET', '/system/info')
}

// Web replacement for native file dialog
export async function SelectFile(title, ext) {
    return new Promise((resolve) => {
        const input = document.createElement('input')
        input.type = 'file'
        if (ext && Array.isArray(ext) && ext.length > 0) {
            input.accept = ext.map((e) => '.' + e.replace(/^\./, '')).join(',')
        }
        input.onchange = async () => {
            if (input.files && input.files[0]) {
                const formData = new FormData()
                formData.append('file', input.files[0])
                try {
                    const resp = await fetch(`${API_BASE}/system/select-file`, {
                        method: 'POST',
                        credentials: 'same-origin',
                        body: formData,
                    })
                    if (resp.status === 401) {
                        resolve(unauthorized())
                        return
                    }
                    resolve(await resp.json())
                } catch {
                    resolve({ success: false, msg: 'upload failed' })
                }
            } else {
                resolve({ success: false, msg: '' })
            }
        }
        input.addEventListener('cancel', () => resolve({ success: false, msg: '' }))
        input.click()
    })
}

export async function SaveFile(title, defaultName, ext) {
    // In web mode, file save dialogs are not applicable
    // The backend ExportLog etc. will handle download differently
    return { success: true, data: { path: '' } }
}

// ==================== Auth Service ====================

export async function Login(username, password) {
    return await call('POST', '/auth/login', { username, password })
}

export async function Logout() {
    return await call('POST', '/auth/logout')
}
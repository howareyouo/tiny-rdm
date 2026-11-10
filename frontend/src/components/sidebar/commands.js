import { get, isEmpty, split } from 'lodash'
import { ref } from 'vue'
import { ClipboardSetText } from 'wailsjs/runtime/runtime.js'
import { nativeRedisKey } from '@/utils/key_convert.js'
import { i18nGlobal } from '@/utils/i18n.js'
import Add from '@/components/icons/Add.vue'
import Refresh from '@/components/icons/Refresh.vue'
import Delete from '@/components/icons/Delete.vue'
import CopyLink from '@/components/icons/CopyLink.vue'
import LoadList from '@/components/icons/LoadList.vue'
import LoadAll from '@/components/icons/LoadAll.vue'
import Unlink from '@/components/icons/Unlink.vue'
import Import from '@/components/icons/Import.vue'
import Export from '@/components/icons/Export.vue'
import Config from '@/components/icons/Config.vue'
import Edit from '@/components/icons/Edit.vue'
import Connect from '@/components/icons/Connect.vue'
import useBrowserStore from 'stores/browser.js'
import useConnectionStore from 'stores/connections.js'
import useDialogStore from 'stores/dialog.js'
import useTabStore from 'stores/tab.js'

/**
 * Command table of the browser tree and the browser function bar.
 *
 * Every command receives `{ server, db, node }`, where `node` is the tree node
 * the command was invoked on or resolved from the current selection.
 *
 * @return {Object}
 */
export function browserCommands() {
    const browserStore = useBrowserStore()
    const dialogStore = useDialogStore()

    const redisKeyOf = (node) => (node == null ? '' : node.redisKeyCode || node.redisKey || '')
    const keyNameOf = (node) => (node != null && node.redisKeyCode ? node.label : redisKeyOf(node))
    const dbOf = (node, db) => get(node, 'db', db)

    const withNodeLoading = (node, task) => {
        if (node == null || node.loading) {
            return undefined
        }
        node.loading = true
        return Promise.resolve(task()).finally(() => delete node.loading)
    }

    const copyKey = (ctx) => {
        ClipboardSetText(nativeRedisKey(redisKeyOf(ctx.node)))
        $message.success(i18nGlobal.t('interface.copy_succ'))
    }

    return {
        key_newkey: {
            label: 'interface.new_key',
            icon: Add,
            run: ({ server, db, node }) =>
                dialogStore.openNewKeyDialog(node == null ? '' : redisKeyOf(node), server, dbOf(node, db)),
        },
        key_reload: {
            label: 'interface.reload',
            icon: Refresh,
            run: ({ server, db, node }) =>
                withNodeLoading(node, () => browserStore.reloadLayer(server, dbOf(node, db), redisKeyOf(node))),
        },
        value_reload: {
            label: 'interface.reload',
            icon: Refresh,
            run: ({ server, db, node }) =>
                browserStore.reloadKey({
                    server,
                    db: dbOf(node, db),
                    key: redisKeyOf(node),
                }),
        },
        db_loadmore: {
            label: 'interface.load_more',
            icon: LoadList,
            run: ({ server, db, node }) =>
                withNodeLoading(node, () =>
                    browserStore.loadMoreKeys(server, dbOf(node, db)).then((fullyLoaded) => {
                        node.fullLoaded = fullyLoaded
                    }),
                ),
        },
        db_loadall: {
            label: 'interface.load_all',
            icon: LoadAll,
            run: ({ server, db, node }) =>
                withNodeLoading(node, () =>
                    browserStore
                        .loadAllKeys(server, dbOf(node, db))
                        .catch((e) => $message.error(e.message))
                        .finally(() => (node.fullLoaded = true)),
                ),
        },
        key_remove: {
            label: 'interface.batch_delete_key',
            icon: Delete,
            run: ({ server, db, node }) => {
                const redisKey = redisKeyOf(node)
                const separator = browserStore.getSeparator(server)
                dialogStore.openDeleteKeyDialog(server, dbOf(node, db), isEmpty(redisKey) ? '*' : redisKey + separator + '*')
            },
        },
        value_remove: {
            label: 'interface.remove_key',
            icon: Delete,
            run: ({ server, db, node }) => {
                const redisKey = redisKeyOf(node)
                $dialog.warning(i18nGlobal.t('dialogue.remove_tip', { name: keyNameOf(node) }), () => {
                    browserStore.deleteKey(server, dbOf(node, db), redisKey).then((success) => {
                        if (success) {
                            $message.success(i18nGlobal.t('dialogue.delete.success', { key: keyNameOf(node) }))
                        }
                    })
                })
            },
        },
        key_copy: {
            label: 'interface.copy_path',
            icon: CopyLink,
            run: copyKey,
        },
        value_copy: {
            label: 'interface.copy_key',
            icon: CopyLink,
            run: copyKey,
        },
        key_import: {
            label: 'interface.import_key',
            icon: Import,
            run: ({ server, db }) => dialogStore.openImportKeyDialog(server, db),
        },
        db_flush: {
            label: 'interface.flush_db',
            icon: Delete,
            run: ({ server, db }) => dialogStore.openFlushDBDialog(server, db),
        },
        db_disconnect: {
            label: 'interface.disconnect',
            icon: Unlink,
            run: ({ server }) => browserStore.closeConnection(server),
        },
    }
}

/**
 * Command table of the connection tree and the connection function bar.
 *
 * Every command receives `{ group, name, node }`, where `group`/`name` are
 * parsed from the key of the node the command was invoked on and `node` is
 * that node itself.
 *
 * @return {{commands: Object, connectingServer: Ref<string>, cancelOpen: () => void}}
 */
export function connectionCommands() {
    const browserStore = useBrowserStore()
    const connectionStore = useConnectionStore()
    const dialogStore = useDialogStore()
    const tabStore = useTabStore()
    const connectingServer = ref('')

    const openConnection = async (name) => {
        try {
            connectingServer.value = name
            if (!browserStore.isConnected(name)) {
                await browserStore.openConnection(name)
            }
            // check if connection already canceled before finish open
            if (!isEmpty(connectingServer.value)) {
                tabStore.upsertTab({
                    server: name,
                    db: browserStore.getSelectedDB(name),
                    forceSwitch: true,
                })
            }
        } catch (e) {
            $message.error(e.message)
        } finally {
            connectingServer.value = ''
        }
    }

    const closeConnection = (name) => {
        browserStore.closeConnection(name).then((closed) => {
            if (closed) {
                $message.success(i18nGlobal.t('dialogue.handle_succ'))
            }
        })
    }

    const editConnection = (name) => {
        // ask for close relevant connections before edit
        if (browserStore.isConnected(name)) {
            $dialog.warning(i18nGlobal.t('dialogue.edit_close_confirm'), () => {
                browserStore.closeConnection(name)
                dialogStore.openEditDialog(name)
            })
        } else {
            dialogStore.openEditDialog(name)
        }
    }

    const removeConnection = (name) => {
        $dialog.warning(
            i18nGlobal.t('dialogue.remove_tip', { type: i18nGlobal.t('dialogue.connection.conn_name'), name }),
            async () => {
                connectionStore.deleteConnection(name).then(({ success, msg }) => {
                    if (!success) {
                        $message.error(msg)
                    }
                })
            },
        )
    }

    const removeGroup = (name) => {
        $dialog.warning(i18nGlobal.t('dialogue.remove_group_tip', { name }), async () => {
            connectionStore.deleteGroup(name).then(({ success, msg }) => {
                if (!success) {
                    $message.error(msg)
                }
            })
        })
    }

    // a node's own name wins over the name parsed from the selection key,
    // so double-click works even when the node is not selected yet
    const nameOf = ({ node, name }) => (node != null && !isEmpty(node.name) ? node.name : name)

    const commands = {
        server_open: {
            label: 'interface.open_connection',
            icon: Connect,
            run: (ctx) => openConnection(nameOf(ctx)),
        },
        server_close: {
            label: 'interface.disconnect',
            icon: Unlink,
            run: (ctx) => closeConnection(nameOf(ctx)),
        },
        server_edit: {
            label: 'interface.edit_conn',
            icon: Config,
            run: (ctx) => editConnection(nameOf(ctx)),
        },
        server_dup: {
            label: 'interface.dup_conn',
            icon: CopyLink,
            run: (ctx) => dialogStore.openDuplicateDialog(nameOf(ctx)),
        },
        server_remove: {
            label: 'interface.remove_conn',
            icon: Delete,
            run: (ctx) => removeConnection(nameOf(ctx)),
        },
        group_rename: {
            label: 'interface.rename_conn_group',
            icon: Edit,
            run: ({ group }) => {
                if (!isEmpty(group)) {
                    dialogStore.openRenameGroupDialog(group)
                }
            },
        },
        group_delete: {
            label: 'interface.remove_conn_group',
            icon: Delete,
            run: ({ group }) => {
                if (!isEmpty(group)) {
                    removeGroup(group)
                }
            },
        },
        conn_import: {
            label: 'interface.import_conn',
            icon: Import,
            run: async () => {
                await connectionStore.importConnections()
                await connectionStore.initConnections(true)
            },
        },
        conn_export: {
            label: 'interface.export_conn',
            icon: Export,
            run: async () => {
                await connectionStore.exportConnections()
            },
        },
    }

    const cancelOpen = () => {
        if (!isEmpty(connectingServer.value)) {
            browserStore.closeConnection(connectingServer.value)
            connectingServer.value = ''
        }
    }

    return { commands, connectingServer, cancelOpen }
}

/**
 * Key of a connection node -> command context.
 *
 * @param {string} key
 * @return {{group: string, name: string}}
 */
export function connectionNodeContext(key) {
    const [group, name] = split(key, '/')
    return { group, name }
}


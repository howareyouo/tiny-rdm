<script setup>
import { computed, h, nextTick, ref, watchEffect } from 'vue'
import { ConnectionType } from '@/consts/connection_type.js'
import { NIcon, NText } from 'naive-ui'
import Key from '@/components/icons/Key.vue'
import Binary from '@/components/icons/Binary.vue'
import Database from '@/components/icons/Database.vue'
import Layer from '@/components/icons/Layer.vue'
import { filter, find, first, get, includes, isEmpty, last, map, size, toUpper } from 'lodash'
import { useI18n } from 'vue-i18n'
import RedisTypeTag from '@/components/common/RedisTypeTag.vue'
import useDialogStore from 'stores/dialog.js'
import useConnectionStore from 'stores/connections.js'
import useTabStore from 'stores/tab.js'
import { parseHexColor } from '@/utils/rgb.js'
import useBrowserStore from 'stores/browser.js'
import { useRender } from '@/utils/render.js'
import usePreferencesStore from 'stores/preferences.js'
import { typesIconStyle } from '@/consts/support_redis_type.js'
import { nativeRedisKey } from '@/utils/key_convert.js'
import { ClipboardSetText } from 'wailsjs/runtime/runtime.js'
import { isMacOS } from '@/utils/platform.js'
import { divider, useTreeShell } from '@/components/sidebar/tree_shell.js'
import { browserCommands } from '@/components/sidebar/commands.js'

const props = defineProps({
    server: String,
    db: Number,
    keyView: String,
    loading: Boolean,
    pattern: String,
    fullLoaded: Boolean,
    checkMode: Boolean,
})

const render = useRender()
const i18n = useI18n()
const connectionStore = useConnectionStore()
const browserStore = useBrowserStore()
const prefStore = usePreferencesStore()
const tabStore = useTabStore()
const dialogStore = useDialogStore()

/**
 *
 * @type {ComputedRef<string[]>}
 */
const expandedKeys = computed(() => {
    const tab = find(tabStore.tabList, { name: props.server })
    if (tab != null) {
        return get(tab, 'expandedKeys', [])
    }
    return []
})

/**
 *
 * @type {ComputedRef<string[]>}
 */
const selectedKeys = computed(() => {
    const tab = find(tabStore.tabList, { name: props.server })
    if (tab != null) {
        return get(tab, 'selectedKeys', [])
    }
    return []
})

/**
 *
 * @type {ComputedRef<string[]>}
 */
const checkedKeys = computed(() => {
    const tab = find(tabStore.tabList, { name: props.server })
    if (tab != null) {
        const checkedKeys = get(tab, 'checkedKeys', [])
        return map(checkedKeys, 'key')
    }
    return []
})

const data = computed(() => {
    return browserStore.getKeyStruct(props.server, props.checkMode)
})

const backgroundColor = computed(() => {
    const { markColor: hex = '' } = connectionStore.serverProfile[props.server] || {}
    if (isEmpty(hex)) {
        return ''
    }
    const { r, g, b } = parseHexColor(hex)
    return `rgba(${r}, ${g}, ${b}, 0.2)`
})

const commands = browserCommands()

const menu = {
    [ConnectionType.RedisKey]: ['key_newkey', 'key_copy', divider('d1'), 'key_remove'],
    [ConnectionType.RedisValue]: ['value_reload', 'value_copy', divider('d1'), 'value_remove'],
}

const buttons = {
    [ConnectionType.RedisDB]: (option) =>
        option.opened === true
            ? [
                  { key: 'db_loadmore', disabled: option.end === true, loading: option.loading === true },
                  { key: 'db_loadall', disabled: option.end === true, loading: option.loading === true },
              ]
            : [],
    [ConnectionType.RedisKey]: (option) => [
        {
            key: 'key_reload',
            label: props.fullLoaded ? 'interface.reload' : 'interface.reload_disable',
            disabled: !props.fullLoaded,
            loading: option.loading === true,
        },
        { key: 'key_newkey' },
        { key: 'key_remove' },
    ],
    [ConnectionType.RedisValue]: ['value_remove'],
}

const {
    contextMenuParam,
    handleSelectContextMenu,
    renderIconMenu,
    renderSuffix: renderSelectedSuffix,
    onNodeContextMenu,
    handleOutsideContextMenu,
    onSelectedKeysUpdate,
    onExpandedKeysUpdate,
} = useTreeShell({
    commands,
    menu,
    buttons,
    context: () => ({ server: props.server, db: props.db }),
    resolveNode: (key) => browserStore.getNode(key),
    selection: {
        get: () => selectedKeys.value,
        set: (keys) => onUpdateSelectedKeys(keys),
    },
    expansion: {
        get: () => expandedKeys.value,
        set: (keys, option, meta) => onUpdateExpanded(keys, option, meta),
    },
})

const handleKeyUp = () => {
    const selectedKey = get(selectedKeys.value, 0)
    if (selectedKey == null) {
        return
    }
    let node = browserStore.getNode(selectedKey)
    if (node == null) {
        return
    }

    let parentNode = browserStore.getParentNode(selectedKey)
    if (parentNode == null) {
        return
    }
    const nodeIndex = parentNode.children.indexOf(node)
    if (nodeIndex <= 0) {
        if (parentNode.type === ConnectionType.RedisKey || parentNode.type === ConnectionType.RedisValue) {
            onUpdateSelectedKeys([parentNode.key])
            updateKeyDetail(parentNode)
        }
        return
    }

    // try select pre node
    let preNode = parentNode.children[nodeIndex - 1]
    while (preNode.expanded && !isEmpty(preNode.children)) {
        preNode = last(preNode.children)
    }
    onUpdateSelectedKeys([preNode.key])
    updateKeyDetail(preNode)
}

const handleKeyDown = () => {
    const selectedKey = get(selectedKeys.value, 0)
    if (selectedKey == null) {
        return
    }
    let node = browserStore.getNode(selectedKey)
    if (node == null) {
        return
    }
    // try select first child if expanded
    if (node.expanded && !isEmpty(node.children)) {
        const childNode = get(node.children, 0)
        onUpdateSelectedKeys([childNode.key])
        updateKeyDetail(childNode)
        return
    }

    let travelCount = 0
    let childKey = selectedKey
    do {
        if (travelCount++ > 20) {
            return
        }
        // find out parent node
        const parentNode = browserStore.getParentNode(childKey)
        if (parentNode == null) {
            return
        }
        const nodeIndex = parentNode.children.indexOf(node)
        if (nodeIndex < 0 || nodeIndex >= parentNode.children.length - 1) {
            // last child, try select parent's neighbor node
            childKey = parentNode.key
            node = parentNode
        } else {
            // select next node
            const childNode = parentNode.children[nodeIndex + 1]
            onUpdateSelectedKeys([childNode.key])
            updateKeyDetail(childNode)
            return
        }
    } while (true)
}

const handleKeyLeft = () => {
    const selectedKey = get(selectedKeys.value, 0)
    if (selectedKey == null) {
        return
    }
    let node = browserStore.getNode(selectedKey)
    if (node == null) {
        return
    }

    if (node.type === ConnectionType.RedisKey) {
        if (node.expanded) {
            // try collapse
            onUpdateExpanded([node.key], null, { node, action: 'collapse' })
            return
        }
    }

    // try select parent node
    let parentNode = browserStore.getParentNode(selectedKey)
    if (parentNode == null || parentNode.type !== ConnectionType.RedisKey) {
        return
    }
    onUpdateSelectedKeys([parentNode.key])
    updateKeyDetail(parentNode)
}

const handleKeyRight = () => {
    const selectedKey = get(selectedKeys.value, 0)
    if (selectedKey == null) {
        return
    }
    let node = browserStore.getNode(selectedKey)
    if (node == null) {
        return
    }

    if (node.type === ConnectionType.RedisKey) {
        if (!node.expanded) {
            // try expand
            onUpdateExpanded([node.key], null, { node, action: 'expand' })
        } else if (!isEmpty(node.children)) {
            // try select first child
            const childNode = first(node.children)
            onUpdateSelectedKeys([childNode.key])
            updateKeyDetail(childNode)
        }
    } else if (node.type === ConnectionType.RedisValue) {
        handleKeyDown()
    }
}

const handleKeyDelete = () => {
    const selectedKey = get(selectedKeys.value, 0)
    if (selectedKey == null) {
        return
    }
    let node = browserStore.getNode(selectedKey)
    if (node == null) {
        return
    }

    const { db = 0, key: nodeKey, redisKey: rk = '', redisKeyCode: rkc, label } = node || {}
    const redisKey = rkc || rk
    const redisKeyName = !!rkc ? label : redisKey
    switch (node.type) {
        case ConnectionType.RedisKey:
            dialogStore.openDeleteKeyDialog(props.server, db, isEmpty(redisKey) ? '*' : redisKey + ':*')
            break
        case ConnectionType.RedisValue:
            $dialog.warning(i18n.t('dialogue.remove_tip', { name: redisKeyName }), () => {
                browserStore.deleteKey(props.server, db, redisKey).then((success) => {
                    if (success) {
                        $message.success(i18n.t('dialogue.delete.success', { key: redisKeyName }))
                    }
                })
            })
            break
    }
}

const handleKeyCopy = () => {
    const selectedKey = get(selectedKeys.value, 0)
    if (selectedKey == null) {
        return
    }
    let node = browserStore.getNode(selectedKey)
    if (node == null) {
        return
    }

    if (node.type === ConnectionType.RedisValue) {
        ClipboardSetText(nativeRedisKey(node.redisKeyCode || node.redisKey))
        $message.success(i18n.t('interface.copy_succ'))
    }
}

const onKeyShortcut = (e) => {
    const isCtrlOn = isMacOS() ? e.metaKey : e.ctrlKey
    switch (e.key) {
        case 'ArrowUp':
            handleKeyUp()
            break
        case 'ArrowDown':
            handleKeyDown()
            break
        case 'ArrowLeft':
            handleKeyLeft()
            break
        case 'ArrowRight':
            handleKeyRight()
            break
        case 'c':
            if (isCtrlOn) {
                handleKeyCopy()
            }
            break
        case 'Delete':
            handleKeyDelete()
            break
        case 'F5':
            handleSelectContextMenu('value_reload')
            break
        case 'r':
            if (isCtrlOn) {
                handleSelectContextMenu('value_reload')
            }
            break
    }
}

const onUpdateSelectedKeys = (keys, options) => {
    if (!isEmpty(keys)) {
        tabStore.setSelectedKeys(props.server, keys)
    } else {
        // default is load blank key to display server status
        // tabStore.openBlank(props.server)
    }
}

const onUpdateExpanded = (value, option, meta) => {
    const expand = meta.action === 'expand'
    if (expand) {
        tabStore.addExpandedKey(props.server, value)
    } else {
        tabStore.removeExpandedKey(props.server, value)
    }
    let node = meta.node
    if (!node) {
        return
    }

    // keep expand or collapse children while they own more than 1 child
    do {
        const key = node.key
        if (expand) {
            if (node.type === ConnectionType.RedisKey) {
                node.expanded = true
                tabStore.addExpandedKey(props.server, key)
            }
        } else {
            node.expanded = false
            tabStore.removeExpandedKey(props.server, key)
        }
        if (size(node.children) === 1) {
            node = node.children[0]
        } else {
            break
        }
    } while (true)
}

/**
 *
 * @param {string[]} keys
 * @param {TreeOption[]} options
 */
const onUpdateCheckedKeys = (keys, options) => {
    const hasFilter = !isEmpty(props.pattern)
    const checkedKeys = map(
        filter(
            options,
            (o) => o.type === ConnectionType.RedisValue && (!hasFilter || includes(o.redisKey, props.pattern)),
        ),
        (o) => ({ key: o.key, redisKey: o.redisKeyCode || o.redisKey }),
    )
    tabStore.setCheckedKeys(props.server, checkedKeys)
}

const renderPrefix = ({ option }) => {
    switch (option.type) {
        case ConnectionType.RedisDB:
            return h(
                NIcon,
                { size: 20, color: option.opened === true ? '#dc423c' : undefined },
                {
                    default: () => h(Database, { inverse: option.opened === true }),
                },
            )

        case ConnectionType.RedisKey:
            return h(
                NIcon,
                { size: 20 },
                {
                    default: () => h(Layer),
                },
            )

        case ConnectionType.RedisValue:
            if (prefStore.keyIconType === typesIconStyle.ICON) {
                return h(NIcon, { size: 20 }, () => h(Key))
            }
            const loading = isEmpty(option.redisType) || option.redisType === 'loading'
            if (loading) {
                browserStore.loadKeyType({
                    server: props.server,
                    db: option.db,
                    key: option.redisKeyCode || option.redisKey,
                })
            }
            switch (prefStore.keyIconType) {
                case typesIconStyle.FULL:
                    return h(RedisTypeTag, {
                        type: toUpper(option.redisType),
                        short: false,
                        size: 'small',
                        inverse: includes(selectedKeys.value, option.key),
                        loading,
                    })

                case typesIconStyle.POINT:
                    return h(RedisTypeTag, {
                        type: toUpper(option.redisType),
                        point: true,
                        loading,
                    })

                case typesIconStyle.SHORT:
                default:
                    return h(RedisTypeTag, {
                        type: toUpper(option.redisType),
                        short: true,
                        size: 'small',
                        loading,
                        inverse: includes(selectedKeys.value, option.key),
                    })
            }
    }
}

// render tree item label
const renderLabel = ({ option }) => {
    switch (option.type) {
        case ConnectionType.RedisKey:
            if (option.label === '') {
                // blank label name
                return h('div', [
                    h(NText, { italic: true, depth: 3 }, () => '[NO NAME]'),
                    h('span', () => ` (${option.keyCount || 0})`),
                ])
            }
            return `${option.label} (${option.keyCount || 0})`
    }
    return option.label
}

// render menu function icon
const renderSuffix = ({ option }) => {
    const selected = includes(selectedKeys.value, option.key)
    if (selected && !props.checkMode) {
        return renderSelectedSuffix({ option })
    }
    if (!selected && !!option.redisKeyCode && option.type === ConnectionType.RedisValue) {
        // render binary icon
        return renderIconMenu(h(NIcon, { size: 20 }, () => h(Binary)))
    }
    return null
}

const lastLoadKey = ref(0)

/**
 *
 * @param {RedisNodeItem} node
 */
const updateKeyDetail = (node) => {
    if (node.type === ConnectionType.RedisValue) {
        const preK = tabStore.getActivatedKey(props.server)
        if (!isEmpty(preK) && preK === node.key && Date.now() - lastLoadKey.value > 1000) {
            // reload key already activated
            lastLoadKey.value = Date.now()
            const { db, redisKey, redisKeyCode } = node
            browserStore.reloadKey({
                server: props.server,
                db,
                key: redisKeyCode || redisKey,
            })
        } else if (tabStore.setActivatedKey(props.server, node.key)) {
            const { db, redisKey, redisKeyCode } = node
            browserStore.loadKeySummary({
                server: props.server,
                db,
                key: redisKeyCode || redisKey,
                clearValue: true,
            })
        }
    }
}

const nodeProps = ({ option }) => {
    return {
        onClick: () => {
            updateKeyDetail(option)
        },
        onDblclick: () => {
            if (props.loading) {
                console.warn('TODO: alert to ignore double click when loading')
                return
            }
            if (!props.checkMode) {
                // default handle is toggle expand current node
                nextTick().then(() => tabStore.toggleExpandKey(props.server, option.key))
            }
        },
        onContextmenu(e) {
            onNodeContextMenu(option, e)
        },
    }
}

watchEffect(
    () => {
        if (!props.checkMode) {
            tabStore.setCheckedKeys(props.server)
        } else {
            tabStore.addExpandedKey(props.server, `${ConnectionType.RedisDB}`)
        }
    },
    { flush: 'post' },
)

// the NTree node may get incorrect height after change data
// add key property for force refresh the component so that everything back to normal
const treeKey = ref(0)
defineExpose({
    handleSelectContextMenu,
    refreshTree: () => {
        treeKey.value = Date.now()
        tabStore.setExpandedKeys(props.server)
        tabStore.setCheckedKeys(props.server)
    },
    deleteCheckedItems: () => {
        const checkedKeys = tabStore.currentCheckedKeys
        const redisKeys = map(checkedKeys, 'redisKey')
        if (!isEmpty(redisKeys)) {
            dialogStore.openDeleteKeyDialog(props.server, props.db, redisKeys)
        }
    },
    exportCheckedItems: () => {
        const checkedKeys = tabStore.currentCheckedKeys
        const redisKeys = map(checkedKeys, 'redisKey')
        if (!isEmpty(redisKeys)) {
            dialogStore.openExportKeyDialog(props.server, props.db, redisKeys)
        }
    },
    updateTTLCheckedItems: () => {
        const checkedKeys = tabStore.currentCheckedKeys
        const redisKeys = map(checkedKeys, 'redisKey')
        if (!isEmpty(redisKeys)) {
            dialogStore.openTTLDialog({
                server: props.server,
                db: props.db,
                keys: redisKeys,
            })
        }
    },
    getSelectedKey: () => {
        return selectedKeys.value || []
    },
})
</script>

<template>
    <div
        :style="{ backgroundColor }"
        class="flex-box-v browser-tree-wrapper"
        @contextmenu="(e) => e.preventDefault()"
        @keydown.esc="contextMenuParam.show = false">
        <n-spin v-if="props.loading" class="fill-height" />
        <n-empty v-else-if="!props.loading && isEmpty(data)" class="empty-content" />
        <n-tree
            v-show="!props.loading && !isEmpty(data)"
            :key="treeKey"
            :animated="false"
            :block-line="true"
            :block-node="true"
            :cancelable="false"
            :cascade="true"
            :checkable="props.checkMode"
            :checked-keys="checkedKeys"
            :data="data"
            :expand-on-click="false"
            :expanded-keys="expandedKeys"
            :filter="(pattern, node) => includes(node.redisKey, pattern)"
            :keyboard="false"
            :node-props="nodeProps"
            :pattern="props.pattern"
            :render-label="renderLabel"
            :render-prefix="renderPrefix"
            :render-suffix="renderSuffix"
            :selected-keys="selectedKeys"
            :show-irrelevant-nodes="false"
            check-strategy="child"
            class="fill-height"
            virtual-scroll
            @keydown="onKeyShortcut"
            @update:selected-keys="onSelectedKeysUpdate"
            @update:expanded-keys="onExpandedKeysUpdate"
            @update:checked-keys="onUpdateCheckedKeys">
            <template #empty>
                <n-empty class="empty-content" />
            </template>
        </n-tree>

        <!-- context menu -->
        <n-dropdown
            :options="contextMenuParam.options"
            :render-icon="({ icon }) => render.renderIcon(icon)"
            :render-label="({ label }) => render.renderLabel($t(label), { class: 'context-menu-item' })"
            :show="contextMenuParam.show"
            :x="contextMenuParam.x"
            :y="contextMenuParam.y"
            placement="bottom-start"
            trigger="manual"
            @clickoutside="handleOutsideContextMenu"
            @select="handleSelectContextMenu" />
    </div>
</template>

<style lang="scss" scoped>
@use '@/styles/content';

.browser-tree-wrapper {
    height: 100%;
    overflow: hidden;
}
</style>

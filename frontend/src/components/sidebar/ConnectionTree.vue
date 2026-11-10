<script setup>
import { h, nextTick, ref } from 'vue'
import { NIcon, NText } from 'naive-ui'
import { ConnectionType } from '@/consts/connection_type.js'
import Folder from '@/components/icons/Folder.vue'
import Server from '@/components/icons/Server.vue'
import Cluster from '@/components/icons/Cluster.vue'
import Config from '@/components/icons/Config.vue'
import { debounce, get, indexOf, isEmpty } from 'lodash'
import { hexGammaCorrection, parseHexColor, toHexColor } from '@/utils/rgb.js'
import useConnectionStore from 'stores/connections.js'
import useBrowserStore from 'stores/browser.js'
import usePreferencesStore from 'stores/preferences.js'
import { useRender } from '@/utils/render.js'
import { divider, useTreeShell } from '@/components/sidebar/tree_shell.js'
import { connectionCommands, connectionNodeContext } from '@/components/sidebar/commands.js'

const render = useRender()
const connectionStore = useConnectionStore()
const browserStore = useBrowserStore()
const prefStore = usePreferencesStore()

const expandedKeys = ref([])
const selectedKeys = ref([])

const props = defineProps({
    filterPattern: {
        type: String,
    },
})

const { commands, connectingServer, cancelOpen: onCancelOpen } = connectionCommands()

const menu = {
    [ConnectionType.Group]: ['group_rename', 'group_delete'],
    [ConnectionType.Server]: (option) =>
        browserStore.isConnected(option.name)
            ? ['server_close', 'server_edit', 'server_dup', divider('d1'), 'server_remove']
            : ['server_open', 'server_edit', 'server_dup', divider('d1'), 'server_remove'],
}

const buttons = {
    [ConnectionType.Group]: [{ key: 'group_rename', icon: Config }, { key: 'group_delete' }],
    [ConnectionType.Server]: (option) => {
        if (browserStore.isConnected(option.name)) {
            return [{ key: 'server_close' }, { key: 'server_edit' }]
        }
        return [{ key: 'server_open' }, { key: 'server_edit' }, { key: 'server_remove' }]
    },
}

const {
    contextMenuParam,
    handleSelectContextMenu,
    renderSuffix,
    onNodeContextMenu,
    onSelectedKeysUpdate,
    onExpandedKeysUpdate,
    run,
    toggleExpand,
} = useTreeShell({
    commands,
    menu,
    buttons,
    context: () => connectionNodeContext(get(selectedKeys.value, 0) || ''),
    selection: {
        get: () => selectedKeys.value,
        set: (keys) => (selectedKeys.value = keys),
    },
    expansion: {
        get: () => expandedKeys.value,
        set: (keys) => (expandedKeys.value = keys),
    },
})

/**
 * get mark color of server saved in preferences
 * @param name
 * @return {null|string}
 */
const getServerMarkColor = (name) => {
    const { markColor = '' } = connectionStore.serverProfile[name] || {}
    if (!isEmpty(markColor)) {
        const rgb = parseHexColor(markColor)
        const rgb2 = hexGammaCorrection(rgb, 0.75)
        return toHexColor(rgb2)
    }
    return null
}

const renderLabel = ({ option }) => {
    if (option.type === ConnectionType.Server) {
        const color = getServerMarkColor(option.name)
        if (color != null) {
            return h(
                NText,
                {
                    style: {
                        color,
                        fontWeight: '450',
                    },
                },
                () => option.label,
            )
        }
    }
    return option.label
}

const renderPrefix = ({ option }) => {
    const iconTransparency = prefStore.isDark ? 0.75 : 1
    switch (option.type) {
        case ConnectionType.Group:
            const opened = indexOf(expandedKeys.value, option.key) !== -1
            return h(
                NIcon,
                { size: 20 },
                {
                    default: () =>
                        h(Folder, {
                            open: opened,
                            fillColor: `rgba(255,206,120,${iconTransparency})`,
                        }),
                },
            )
        case ConnectionType.Server:
            const connected = browserStore.isConnected(option.name)
            const color = getServerMarkColor(option.name)
            const icon = option.cluster === true ? Cluster : Server
            return h(
                NIcon,
                { size: 20, color: !!!connected ? color : '#dc423c' },
                {
                    default: () =>
                        h(icon, {
                            inverse: !!connected,
                            fillColor: `rgba(220,66,60,${iconTransparency})`,
                        }),
                },
            )
    }
}

const nodeProps = ({ option }) => {
    return {
        onDblclick: async () => {
            if (option.type === ConnectionType.Server) {
                run('server_open', option)
            } else if (option.type === ConnectionType.Group) {
                // toggle expand
                nextTick().then(() => toggleExpand(option))
            }
        },
        onContextmenu(e) {
            onNodeContextMenu(option, e)
        },
    }
}

const findSiblingsAndIndex = (node, nodes) => {
    if (!nodes) {
        return [null, null]
    }
    for (let i = 0; i < nodes.length; ++i) {
        const siblingNode = nodes[i]
        if (siblingNode.key === node.key) {
            return [nodes, i]
        }
        const [siblings, index] = findSiblingsAndIndex(node, siblingNode.children)
        if (siblings && index !== null) {
            return [siblings, index]
        }
    }
    return [null, null]
}

const isRootNode = (node) => {
    return connectionStore.connections.some((item) => item.key === node.key)
}

// delay save until drop stopped after 2 seconds
const saveSort = debounce(connectionStore.saveConnectionSorted, 1500, { trailing: true })
const handleDrop = ({ node, dragNode, dropPosition }) => {
    const [dragNodeSiblings, dragNodeIndex] = findSiblingsAndIndex(dragNode, connectionStore.connections)
    if (dragNodeSiblings === null || dragNodeIndex === null) {
        return
    }
    if (node.type === ConnectionType.Group && dragNode.type === ConnectionType.Group) {
        return
    }
    if (dragNode.type === ConnectionType.Group && !isRootNode(node)) {
        return
    }
    dragNodeSiblings.splice(dragNodeIndex, 1)
    if (dropPosition === 'inside') {
        if (node.children) {
            node.children.unshift(dragNode)
        } else {
            node.children = [dragNode]
        }
    } else if (dropPosition === 'before') {
        const [nodeSiblings, nodeIndex] = findSiblingsAndIndex(node, connectionStore.connections)
        if (nodeSiblings === null || nodeIndex === null) {
            return
        }
        nodeSiblings.splice(nodeIndex, 0, dragNode)
    } else if (dropPosition === 'after') {
        const [nodeSiblings, nodeIndex] = findSiblingsAndIndex(node, connectionStore.connections)
        if (nodeSiblings === null || nodeIndex === null) {
            return
        }
        nodeSiblings.splice(nodeIndex + 1, 0, dragNode)
    }
    connectionStore.connections = Array.from(connectionStore.connections)
    saveSort()
}
</script>

<template>
    <div class="connection-tree-wrapper" @keydown.esc="contextMenuParam.show = false">
        <n-tree
            :animated="false"
            :block-line="true"
            :block-node="true"
            :cancelable="false"
            :data="connectionStore.connections"
            :draggable="true"
            :expanded-keys="expandedKeys"
            :node-props="nodeProps"
            :pattern="props.filterPattern"
            :render-label="renderLabel"
            :render-prefix="renderPrefix"
            :render-suffix="renderSuffix"
            :selected-keys="selectedKeys"
            class="fill-height"
            virtual-scroll
            @drop="handleDrop"
            @update:selected-keys="onSelectedKeysUpdate"
            @update:expanded-keys="onExpandedKeysUpdate">
            <template #empty>
                <n-empty :description="$t('interface.empty_server_list')" class="empty-content" />
            </template>
        </n-tree>

        <!-- status display modal -->
        <n-modal :show="connectingServer !== ''" transform-origin="center">
            <n-card
                :bordered="false"
                :content-style="{ textAlign: 'center' }"
                aria-model="true"
                role="dialog"
                style="width: 400px">
                <n-spin>
                    <template #description>
                        <n-space vertical>
                            <n-text strong>{{ $t('dialogue.opening_connection') }}</n-text>
                            <n-button :focusable="false" secondary size="small" @click="onCancelOpen">
                                {{ $t('dialogue.interrupt_connection') }}
                            </n-button>
                        </n-space>
                    </template>
                </n-spin>
            </n-card>
        </n-modal>

        <!-- context menu -->
        <n-dropdown
            :keyboard="true"
            :options="contextMenuParam.options"
            :render-icon="({ icon }) => render.renderIcon(icon)"
            :render-label="({ label }) => render.renderLabel($t(label), { class: 'context-menu-item' })"
            :show="contextMenuParam.show"
            :x="contextMenuParam.x"
            :y="contextMenuParam.y"
            placement="bottom-start"
            trigger="manual"
            @clickoutside="contextMenuParam.show = false"
            @select="handleSelectContextMenu" />
    </div>
</template>

<style lang="scss" scoped>
@use '@/styles/content';

.connection-tree-wrapper {
    height: 100%;
    overflow: hidden;
}
</style>

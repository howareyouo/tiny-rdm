import { h, markRaw, nextTick, reactive } from 'vue'
import { NSpace, useThemeVars } from 'naive-ui'
import IconButton from '@/components/common/IconButton.vue'
import { filter, get, includes, isEmpty, map, union } from 'lodash'

/**
 * A menu entry that is not a command.
 *
 * @param {string} key
 */
export const divider = (key) => ({ type: 'divider', key })

/**
 * Build dropdown options from a list of command keys.
 *
 * @param {Object} commands
 * @param {(string|Object)[]} specs
 * @return {Object[]}
 */
export function buildCommandMenu(commands, specs) {
    return map(specs, (spec) => {
        if (typeof spec !== 'string') {
            return spec
        }
        const command = commands[spec]
        return { key: spec, label: command.label, icon: command.icon }
    })
}

/**
 * Dispatch a domain command by key.
 *
 * @param {Object} commands
 * @param {string} key
 * @param {Object} [ctx]
 */
function runCommand(commands, key, ctx = {}) {
    const command = commands[key]
    if (command == null) {
        console.warn('TODO: handle command:' + key)
        return
    }
    return command.run(ctx)
}

/**
 * Shared machinery of the sidebar trees: context menu, node suffix buttons,
 * selection/expansion setters and command dispatch.
 *
 * Node vocabulary (labels, icons, which commands apply, menu order) stays in
 * each tree through its own `menu` and `buttons` tables.
 *
 * @param {Object} options
 * @param {Object} options.commands command table of the domain
 * @param {Object} options.menu node type -> command keys
 * @param {Object} options.buttons node type -> button specs
 * @param {() => Object} [options.context] values merged into every command ctx
 * @param {(key: string) => Object} [options.resolveNode] node lookup by key
 * @param {{get: () => string[], set: (keys: string[]) => void}} options.selection
 * @param {{get: () => string[], set: (keys: string[], option: Object, meta: Object) => void}} [options.expansion]
 */
export function useTreeShell({ commands, menu, buttons, context, resolveNode, selection, expansion }) {
    const themeVars = useThemeVars()

    const contextMenuParam = reactive({
        show: false,
        x: 0,
        y: 0,
        options: null,
        currentNode: null,
    })

    const commandContext = (node) => ({ ...(context ? context() : {}), node })

    const selectedKeys = () => selection.get() || []

    const selectedNode = () => {
        const selectedKey = get(selectedKeys(), 0)
        if (selectedKey == null || resolveNode == null) {
            return null
        }
        return resolveNode(selectedKey)
    }

    const run = (key, node) => runCommand(commands, key, commandContext(node))

    const specsOf = (table, option) => {
        const specs = table[option.type]
        if (specs == null) {
            return null
        }
        return typeof specs === 'function' ? specs(option, commandContext(option)) : specs
    }

    const buildButton = (option, spec) => {
        const { key, label, icon, disabled, loading } = typeof spec === 'string' ? { key: spec } : spec
        const command = commands[key]
        return h(IconButton, {
            tTooltip: label || command.label,
            icon: icon || command.icon,
            disabled: disabled === true,
            loading: loading === true,
            color: loading === true ? themeVars.value.primaryColor : '',
            onClick: () => run(key, option),
        })
    }

    const renderIconMenu = (items) => {
        return h(
            NSpace,
            {
                align: 'center',
                inline: true,
                size: 3,
                wrapItem: false,
                wrap: false,
                style: 'margin-right: 5px',
            },
            () => items,
        )
    }

    const renderSuffix = ({ option }) => {
        if (!includes(selectedKeys(), option.key)) {
            return null
        }
        const specs = specsOf(buttons, option)
        if (isEmpty(specs)) {
            return null
        }
        return renderIconMenu(map(specs, (spec) => buildButton(option, spec)))
    }

    const onNodeContextMenu = (option, e) => {
        e.preventDefault()
        const specs = specsOf(menu, option)
        if (specs == null) {
            return
        }
        contextMenuParam.show = false
        nextTick().then(() => {
            contextMenuParam.options = markRaw(buildCommandMenu(commands, specs))
            contextMenuParam.currentNode = option
            contextMenuParam.x = e.clientX
            contextMenuParam.y = e.clientY
            contextMenuParam.show = true
            selection.set([option.key])
        })
    }

    const handleOutsideContextMenu = () => {
        contextMenuParam.show = false
    }

    const handleSelectContextMenu = (key) => {
        contextMenuParam.show = false
        if (get(selectedKeys(), 0) == null) {
            return
        }
        run(key, selectedNode())
    }

    const onSelectedKeysUpdate = (keys) => {
        selection.set(keys)
    }

    const onExpandedKeysUpdate = (keys, option, meta) => {
        if (expansion == null) {
            return
        }
        expansion.set(keys, option, meta)
    }

    const setNodeExpanded = (option, expand) => {
        if (expansion == null) {
            return
        }
        const keys = expansion.get() || []
        const next = expand ? union(keys, [option.key]) : filter(keys, (key) => key !== option.key)
        expansion.set(next, option, { node: option, action: expand ? 'expand' : 'collapse' })
    }

    const toggleExpand = (option) => {
        if (expansion == null) {
            return
        }
        setNodeExpanded(option, !includes(expansion.get() || [], option.key))
    }

    return {
        contextMenuParam,
        selectedKeys,
        selectedNode,
        run,
        renderIconMenu,
        renderSuffix,
        onNodeContextMenu,
        handleOutsideContextMenu,
        handleSelectContextMenu,
        onSelectedKeysUpdate,
        onExpandedKeysUpdate,
        setNodeExpanded,
        toggleExpand,
    }
}
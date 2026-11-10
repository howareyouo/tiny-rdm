<script setup>
import { computed, h } from 'vue'
import { useI18n } from 'vue-i18n'
import { isEmpty } from 'lodash'
import useDialogStore from 'stores/dialog.js'
import { types as redisTypes } from '@/consts/support_redis_type.js'
import { decodeTypes, formatTypes } from '@/consts/value_view_type.js'
import useBrowserStore from 'stores/browser.js'
import EntryTable from '@/components/content_value/EntryTable.vue'
import { decodeRedisKey } from '@/utils/key_convert.js'
import { TextAlignType } from '@/consts/text_align_type.js'

const i18n = useI18n()

const props = defineProps({
    name: String,
    db: Number,
    keyPath: String,
    keyCode: {
        type: Array,
        default: null,
    },
    ttl: {
        type: Number,
        default: -1,
    },
    value: {
        type: [String, Array],
        default: () => [],
    },
    size: Number,
    length: Number,
    format: String,
    decode: String,
    end: Boolean,
    loading: Boolean,
    textAlign: Number,
})

const emit = defineEmits(['loadmore', 'loadall', 'reload', 'match', 'update:textAlign'])

/**
 *
 * @type {ComputedRef<string|number[]>}
 */
const keyName = computed(() => {
    return !isEmpty(props.keyCode) ? props.keyCode : props.keyPath
})

const browserStore = useBrowserStore()
const dialogStore = useDialogStore()

const fieldColumn = computed(() => ({
    key: 'key',
    title: () => i18n.t('common.field'),
    align: props.textAlign !== TextAlignType.Left ? 'center' : 'left',
    titleAlign: 'center',
    resizable: true,
    sorter: (row1, row2) =>
        `${row1?.k || ''}`.localeCompare(`${row2?.k || ''}`, undefined, {
            numeric: true,
            sensitivity: 'base',
        }),
    ellipsis: {
        tooltip: {
            style: {
                maxWidth: '50vw',
                maxHeight: '50vh',
            },
            scrollable: true,
        },
        lineClamp: 1,
    },
    className: 'wordline',
    filter: (value, row) => {
        if (isEmpty(value)) {
            return true
        }
        return !!~row.k.indexOf(value.toString())
    },
    render: (row) => {
        if (row.rm === true) {
            return h('s', {}, decodeRedisKey(row.k))
        }
        return decodeRedisKey(row.k)
    },
}))

const findValueIndexByKey = (fieldKey) => {
    if (!(props.value instanceof Array)) {
        return -1
    }
    return props.value.findIndex((item) => item.k === fieldKey)
}

const onRefresh = async (row) => {
    const { updated, success, msg } = await browserStore.getHashField({
        server: props.name,
        db: props.db,
        key: keyName.value,
        field: row.k,
        decode: props.decode,
        format: props.format,
    })
    if (success) {
        const rowIndex = findValueIndexByKey(row.k)
        if (rowIndex >= 0) {
            delete props.value[rowIndex]['rm']
        }
        $message.success(i18n.t('dialogue.reload_succ'))
    } else {
        // update fail, the key may have been deleted
        $message.error(msg)
        const rowIndex = findValueIndexByKey(row.k)
        if (rowIndex >= 0) {
            props.value[rowIndex]['rm'] = true
        }
    }
}

const onSave = async (row, field, value, decode, format) => {
    try {
        if (row == null) {
            throw new Error('row not exists')
        }
        if (isEmpty(field)) {
            field = row.k
        }
        const rowIndex = findValueIndexByKey(row.k)
        const { success, msg } = await browserStore.setHash({
            server: props.name,
            db: props.db,
            key: keyName.value,
            field: row.k,
            newField: field,
            value,
            decode,
            format,
            retDecode: props.decode,
            retFormat: props.format,
            index: rowIndex >= 0 ? [rowIndex] : undefined,
        })
        if (success) {
            $message.success(i18n.t('interface.save_value_succ'))
        } else {
            $message.error(msg)
        }
    } catch (e) {
        $message.error(e.message)
    }
}

const onDelete = async (row) => {
    try {
        const { removed, success, msg } = await browserStore.removeHashField({
            server: props.name,
            db: props.db,
            key: keyName.value,
            field: row.k,
            reload: false,
        })
        if (success) {
            const rowIndex = findValueIndexByKey(row.k)
            if (rowIndex >= 0) {
                props.value.splice(rowIndex, 1)
            }
            $message.success(i18n.t('dialogue.delete.success', { key: row.k }))
        } else {
            $message.error(msg)
        }
    } catch (e) {
        $message.error(e.message)
    }
}

const onAddRow = () => {
    dialogStore.openAddFieldsDialog(props.name, props.db, props.keyPath, props.keyCode, redisTypes.HASH)
}
</script>

<template>
    <div class="content-wrapper flex-box-v">
        <slot name="toolbar" />
        <entry-table
            :bind-key="(row) => row.k"
            :decode="props.decode"
            :editable="true"
            :end="props.end"
            :entries="props.value"
            :edit-column-key="'key'"
            :edit-field="(row) => row.k"
            :field-label="$t('common.field')"
            :fields="[fieldColumn]"
            :format="props.format"
            :key-path="props.keyPath"
            :length="props.length"
            :loading="props.loading"
            :mem-size="props.size"
            :refreshable="true"
            :row-key="'k'"
            :text-align="props.textAlign"
            :value-last="true"
            @add="onAddRow"
            @delete="onDelete"
            @loadall="emit('loadall')"
            @loadmore="emit('loadmore')"
            @match="(val) => emit('match', val)"
            @refresh="onRefresh"
            @reload="(decode, format) => emit('reload', decode, format)"
            @save="onSave"
            @update:text-align="(val) => emit('update:textAlign', val)" />
    </div>
</template>

<style lang="scss" scoped></style>

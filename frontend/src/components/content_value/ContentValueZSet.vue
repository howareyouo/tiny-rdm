<script setup>
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { isEmpty } from 'lodash'
import useDialogStore from 'stores/dialog.js'
import { types as redisTypes } from '@/consts/support_redis_type.js'
import { decodeTypes, formatTypes } from '@/consts/value_view_type.js'
import useBrowserStore from 'stores/browser.js'
import EntryTable from '@/components/content_value/EntryTable.vue'
import { nativeRedisKey } from '@/utils/key_convert.js'
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
        type: Array,
        default: () => [],
    },
    size: Number,
    length: Number,
    format: {
        type: String,
        default: formatTypes.RAW,
    },
    decode: {
        type: String,
        default: decodeTypes.NONE,
    },
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

const scoreColumn = computed(() => ({
    key: 'score',
    title: () => i18n.t('common.score'),
    align: props.textAlign !== TextAlignType.Left ? 'center' : 'left',
    titleAlign: 'center',
    resizable: true,
    sorter: (row1, row2) => row1.s - row2.s,
    render: (row) => {
        return row.ss || row.s
    },
}))

const valueFilter = (filterValue, row) => {
    const val = row.dv || nativeRedisKey(row.v)
    return !!~val.indexOf(filterValue.toString())
}

const onSave = async (row, field, value, decode, format) => {
    try {
        const score = parseFloat(field)
        if (row == null) {
            throw new Error('row not exists')
        }
        if (isEmpty(value)) {
            value = row.v
        }
        const { success, msg } = await browserStore.updateZSetItem({
            server: props.name,
            db: props.db,
            key: keyName.value,
            value: row.v,
            newValue: value,
            score,
            decode,
            format,
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
        const { success, msg } = await browserStore.removeZSetItem({
            server: props.name,
            db: props.db,
            key: keyName.value,
            value: row.v,
        })
        if (success) {
            $message.success(i18n.t('dialogue.delete.success', { key: row.v }))
        } else {
            $message.error(msg)
        }
    } catch (e) {
        $message.error(e.message)
    }
}

const onAddRow = () => {
    dialogStore.openAddFieldsDialog(props.name, props.db, props.keyPath, props.keyCode, redisTypes.ZSET)
}
</script>

<template>
    <div class="content-wrapper flex-box-v">
        <slot name="toolbar" />
        <entry-table
            :decode="props.decode"
            :editable="true"
            :end="props.end"
            :entries="props.value"
            :edit-field="(row) => row.s"
            :field-label="$t('common.score')"
            :fields="[scoreColumn]"
            :format="props.format"
            :key-path="props.keyPath"
            :length="props.length"
            :loading="props.loading"
            :mem-size="props.size"
            :text-align="props.textAlign"
            :value-filter="valueFilter"
            @add="onAddRow"
            @delete="onDelete"
            @loadall="emit('loadall')"
            @loadmore="emit('loadmore')"
            @match="(val) => emit('match', val)"
            @reload="(decode, format) => emit('reload', decode, format)"
            @save="onSave"
            @update:text-align="(val) => emit('update:textAlign', val)" />
    </div>
</template>

<style lang="scss" scoped></style>

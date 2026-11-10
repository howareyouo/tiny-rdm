<script setup>
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { isEmpty } from 'lodash'
import { types as redisTypes } from '@/consts/support_redis_type.js'
import useDialogStore from 'stores/dialog.js'
import { decodeTypes, formatTypes } from '@/consts/value_view_type.js'
import useBrowserStore from 'stores/browser.js'
import EntryTable from '@/components/content_value/EntryTable.vue'
import { nativeRedisKey } from '@/utils/key_convert.js'

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
        // [{v: string|number[], dv: string}]
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

const valueFilter = (filterValue, row) => {
    if (isEmpty(filterValue)) {
        return true
    }
    const val = row.dv || nativeRedisKey(row.v)
    return !!~val.indexOf(filterValue.toString())
}

const onSave = async (row, field, value, decode, format) => {
    try {
        if (row == null) {
            throw new Error('row not exists')
        }
        if (isEmpty(value)) {
            value = row.v
        }
        const { success, msg } = await browserStore.updateListItem({
            server: props.name,
            db: props.db,
            key: keyName.value,
            index: row.index,
            value,
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

const onDelete = async (row, index) => {
    try {
        const { success, msg } = await browserStore.removeListItem({
            server: props.name,
            db: props.db,
            key: keyName.value,
            index,
        })
        if (success) {
            $message.success(i18n.t('dialogue.delete.success', { key: `#${index + 1}` }))
        } else {
            $message.error(msg)
        }
    } catch (e) {
        $message.error(e.message)
    }
}

const onAddValue = () => {
    dialogStore.openAddFieldsDialog(props.name, props.db, props.keyPath, props.keyCode, redisTypes.LIST)
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
            :field-label="$t('common.index')"
            :field-readonly="true"
            :fields="[]"
            :format="props.format"
            :key-path="props.keyPath"
            :length="props.length"
            :loading="props.loading"
            :mem-size="props.size"
            :row-index="(row) => row.index"
            :text-align="props.textAlign"
            :value-filter="valueFilter"
            @add="onAddValue"
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

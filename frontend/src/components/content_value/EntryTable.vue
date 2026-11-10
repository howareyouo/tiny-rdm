<script setup>
import { computed, h, reactive, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { NIcon, useThemeVars } from 'naive-ui'
import AddLink from '@/components/icons/AddLink.vue'
import EditableTableColumn from '@/components/common/EditableTableColumn.vue'
import { size, truncate } from 'lodash'
import { decodeTypes, formatTypes } from '@/consts/value_view_type.js'
import LoadList from '@/components/icons/LoadList.vue'
import LoadAll from '@/components/icons/LoadAll.vue'
import IconButton from '@/components/common/IconButton.vue'
import ContentEntryEditor from '@/components/content_value/ContentEntryEditor.vue'
import Edit from '@/components/icons/Edit.vue'
import FormatSelector from '@/components/content_value/FormatSelector.vue'
import ContentSearchInput from '@/components/content_value/ContentSearchInput.vue'
import { formatBytes } from '@/utils/byte_convert.js'
import { ClipboardSetText } from 'wailsjs/runtime/runtime.js'
import SwitchButton from '@/components/common/SwitchButton.vue'
import AlignLeft from '@/components/icons/AlignLeft.vue'
import AlignCenter from '@/components/icons/AlignCenter.vue'
import { TextAlignType } from '@/consts/text_align_type.js'
import { nativeRedisKey } from '@/utils/key_convert.js'

const props = defineProps({
    entries: {
        type: Array,
        default: () => [],
    },
    fields: {
        type: Array,
        default: () => [],
    },
    valueLast: Boolean,
    filterKey: {
        type: String,
        default: 'value',
    },
    valueFilter: Function,
    codeValue: Boolean,
    rowKey: {
        type: String,
        default: 'v',
    },
    rowIndex: {
        type: Function,
        default: (row, index) => index,
    },
    bindKey: {
        type: Function,
        default: (row) => row.v,
    },
    copyText: {
        type: Function,
        default: (row) => row.v,
    },
    editField: {
        type: Function,
        default: (row, index) => index + 1,
    },
    editColumnKey: {
        type: String,
        default: 'value',
    },
    editable: {
        type: Boolean,
        default: true,
    },
    fieldLabel: String,
    fieldReadonly: Boolean,
    refreshable: Boolean,
    actionWidth: {
        type: Number,
        default: 120,
    },
    keyPath: String,
    length: Number,
    memSize: Number,
    loading: Boolean,
    end: Boolean,
    format: String,
    decode: String,
    textAlign: Number,
    showAlign: {
        type: Boolean,
        default: true,
    },
})

const emit = defineEmits([
    'add',
    'delete',
    'refresh',
    'save',
    'match',
    'loadmore',
    'loadall',
    'reload',
    'update:textAlign',
])

const i18n = useI18n()
const themeVars = useThemeVars()

const searchInputRef = ref(null)
const filterOption = ref(null)

const editRow = reactive({
    no: 0,
    field: null,
    value: null,
    format: formatTypes.RAW,
    decode: decodeTypes.NONE,
})
let editingRow = null

const inEdit = computed(() => {
    return editRow.no > 0
})
const fullEdit = ref(false)

const isCode = computed(() => {
    return props.codeValue || props.format === formatTypes.JSON || props.format === formatTypes.UNICODE_JSON
})

const withFilter = (column) => {
    if (column.key !== props.filterKey) {
        return column
    }
    return { ...column, filterOptionValue: filterOption.value }
}

const valueColumn = computed(() => ({
    key: 'value',
    title: () => i18n.t('common.value'),
    align: isCode.value ? 'left' : props.textAlign !== TextAlignType.Left ? 'center' : 'left',
    titleAlign: 'center',
    resizable: true,
    ellipsis: isCode.value
        ? false
        : {
              tooltip: {
                  style: {
                      maxWidth: '50vw',
                      maxHeight: '50vh',
                  },
                  scrollable: true,
              },
              lineClamp: 1,
          },
    className: inEdit.value ? 'clickable' : '',
    filter: props.valueFilter,
    render: (row) => {
        if (isCode.value) {
            const val = row.dv || nativeRedisKey(row.v)
            return h('pre', { class: 'pre-wrap' }, val)
        }
        let val = row.dv || nativeRedisKey(row.v, 500)
        val = truncate(val, { length: 500 })
        if (row.rm === true) {
            return h('s', {}, val)
        }
        return val
    },
}))

const noColumn = computed(() => ({
    key: 'no',
    title: '#',
    width: 80,
    align: 'center',
    titleAlign: 'center',
    render: (row, index) => {
        if (inEdit.value && props.rowIndex(row, index) + 1 === editRow.no) {
            return h(NIcon, { size: 16, color: 'red' }, () => h(Edit, { strokeWidth: 5 }))
        }
        return props.rowIndex(row, index) + 1
    },
}))

const actionColumn = computed(() => ({
    key: 'action',
    title: () => i18n.t('interface.action'),
    width: props.actionWidth,
    align: 'center',
    titleAlign: 'center',
    fixed: 'right',
    render: (row, index) => {
        const idx = props.rowIndex(row, index)
        return h(EditableTableColumn, {
            editing: false,
            bindKey: props.bindKey(row, idx),
            canRefresh: props.refreshable,
            readonly: !props.editable,
            onRefresh: () => emit('refresh', row, idx),
            onCopy: async () => {
                await ClipboardSetText(props.copyText(row))
                $message.success(i18n.t('interface.copy_succ'))
            },
            onEdit: () => startEdit(row, idx),
            onDelete: async () => emit('delete', row, idx),
        })
    },
}))

const columns = computed(() => {
    if (inEdit.value) {
        const list = [noColumn.value]
        if (props.editColumnKey === 'value') {
            list.push(withFilter(valueColumn.value))
            return list
        }
        const field = props.fields.find((column) => column.key === props.editColumnKey)
        if (field != null) {
            list.push(withFilter({ ...field, className: `${field.className || ''} clickable`.trim() }))
        }
        return list
    }
    const fields = props.fields.map(withFilter)
    const value = withFilter(valueColumn.value)
    if (props.valueLast) {
        return [noColumn.value, ...fields, value, actionColumn.value]
    }
    return [noColumn.value, value, ...fields, actionColumn.value]
})

const rowProps = (row, index) => {
    return {
        onClick: () => {
            if (inEdit.value) {
                startEdit(row, props.rowIndex(row, index))
            }
        },
    }
}

const startEdit = (row, index) => {
    if (!props.editable) {
        return
    }
    editingRow = row
    editRow.field = props.editField(row, index)
    editRow.value = row.v
    editRow.decode = props.decode
    editRow.format = props.format
    editRow.no = index + 1
}

const resetEdit = () => {
    editRow.no = 0
    editRow.field = null
    editRow.value = null
    editingRow = null
}

const onSave = (field, value, decode, format) => {
    emit('save', editingRow, field, value, decode, format)
}

const entriesText = computed(() => {
    const len = size(props.entries)
    return `${len} / ${Math.max(len, props.length)}`
})

const loadProgress = computed(() => {
    const len = size(props.entries)
    return (len * 100) / Math.max(len, props.length)
})

const showMemoryUsage = computed(() => {
    return !isNaN(props.memSize) && props.memSize > 0
})

const onFilterInput = (val) => {
    filterOption.value = val
}

const onMatchInput = (matchVal, filterVal) => {
    filterOption.value = filterVal
    emit('match', matchVal)
}

const onUpdateFilter = (filters, sourceColumn) => {
    filterOption.value = filters[sourceColumn.key]
}

const onFormatChanged = (selDecode, selFormat) => {
    emit('reload', selDecode, selFormat)
}

defineExpose({
    reset: () => {
        resetEdit()
        searchInputRef.value?.reset()
    },
})
</script>

<template>
    <div class="tb2 value-item-part flex-box-h">
        <div class="flex-box-h" style="max-width: 50%">
            <content-search-input
                ref="searchInputRef"
                @filter-changed="onFilterInput"
                @match-changed="onMatchInput" />
        </div>
        <div class="flex-item-expand"></div>
        <switch-button
            v-if="props.showAlign"
            :icons="[AlignCenter, AlignLeft]"
            :stroke-width="3.5"
            :t-tooltips="['interface.text_align_center', 'interface.text_align_left']"
            :value="props.textAlign"
            size="medium"
            unselect-stroke-width="3"
            @update:value="(val) => emit('update:textAlign', val)" />
        <n-divider v-if="props.showAlign" vertical />
        <n-button-group>
            <icon-button
                :disabled="props.end || props.loading"
                :icon="LoadList"
                border
                size="18"
                t-tooltip="interface.load_more_entries"
                @click="emit('loadmore')" />
            <icon-button
                :disabled="props.end || props.loading"
                :icon="LoadAll"
                border
                size="18"
                t-tooltip="interface.load_all_entries"
                @click="emit('loadall')" />
        </n-button-group>
        <n-button :focusable="false" plain @click="emit('add')">
            <template #icon>
                <n-icon :component="AddLink" size="18" />
            </template>
            {{ $t('interface.add_row') }}
        </n-button>
    </div>
    <!-- loaded progress -->
    <n-progress
        :border-radius="0"
        :color="props.end ? '#0000' : themeVars.primaryColor"
        :height="2"
        :percentage="loadProgress"
        :processing="props.loading"
        :show-indicator="false"
        status="success"
        type="line" />
    <div class="value-wrapper value-item-part flex-box-h flex-item-expand">
        <!-- table -->
        <n-data-table
            :bordered="false"
            :bottom-bordered="false"
            :columns="columns"
            :data="props.entries"
            :loading="props.loading"
            :row-key="(row) => row[props.rowKey]"
            :row-props="rowProps"
            :single-column="true"
            :single-line="false"
            class="flex-item-expand"
            flex-height
            size="small"
            striped
            virtual-scroll
            @update:filters="onUpdateFilter" />

        <!-- edit pane -->
        <div
            v-if="props.editable"
            v-show="inEdit"
            :style="{ position: fullEdit ? 'static' : 'relative' }"
            class="entry-editor-container flex-item-expand"
            style="width: 100%">
            <content-entry-editor
                v-model:decode="editRow.decode"
                v-model:format="editRow.format"
                v-model:fullscreen="fullEdit"
                :field="editRow.field"
                :field-label="props.fieldLabel"
                :field-readonly="props.fieldReadonly"
                :key-path="props.keyPath"
                :show="inEdit"
                :value="editRow.value"
                :value-label="$t('common.value')"
                class="flex-item-expand"
                style="width: 100%"
                @close="resetEdit"
                @save="onSave" />
        </div>
    </div>
    <div class="value-footer flex-box-h">
        <n-text v-if="!isNaN(props.length)">{{ $t('interface.entries') }}: {{ entriesText }}</n-text>
        <n-divider v-if="showMemoryUsage" vertical />
        <n-text v-if="showMemoryUsage">{{ $t('interface.memory_usage') }}: {{ formatBytes(props.memSize) }}</n-text>
        <div class="flex-item-expand"></div>
        <format-selector
            v-show="!inEdit"
            :decode="props.decode"
            :disabled="inEdit"
            :format="props.format"
            @format-changed="onFormatChanged" />
    </div>
</template>

<style lang="scss" scoped>
.value-footer {
    border-top: v-bind('themeVars.borderColor') 1px solid;
    background-color: v-bind('themeVars.tableHeaderColor');
}
</style>
<script setup lang="ts" generic="T">
import {
  FlexRender,
  getCoreRowModel,
  useVueTable,
  type ColumnDef,
  type TableOptions,
} from "@tanstack/vue-table";

import Pagination from "@/components/sag-ui/pagination/Pagination.vue";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";

interface PaginationProps {
  page: number;
  pageSize: number;
  total: number;
  pageSizeOptions?: number[];
  onPageChange: (page: number) => void;
  onPageSizeChange: (pageSize: number) => void;
}

const props = withDefaults(
  defineProps<{
    columns: ColumnDef<T, any>[];
    data: T[];
    loading?: boolean;
    emptyText?: string;
    pagination?: PaginationProps;
    onRowClick?: (row: T) => void;
    rowClassName?: (row: T) => string;
    tableOptions?: Omit<Partial<TableOptions<T>>, "data" | "columns" | "getCoreRowModel">;
  }>(),
  {
    loading: false,
    emptyText: "暂无数据",
  },
);

const table = useVueTable({
  get data() {
    return props.data;
  },
  columns: props.columns,
  getCoreRowModel: getCoreRowModel(),
  ...(props.pagination
    ? {
        manualPagination: true,
        pageCount: Math.ceil(props.pagination.total / props.pagination.pageSize),
      }
    : {}),
  ...props.tableOptions,
} as TableOptions<T>);
</script>

<template>
  <div class="flex flex-col gap-4">
    <div class="overflow-hidden rounded-lg border">
      <Table>
        <TableHeader class="bg-muted sticky top-0 z-10">
          <TableRow v-for="headerGroup in table.getHeaderGroups()" :key="headerGroup.id">
            <TableHead v-for="header in headerGroup.headers" :key="header.id" :colspan="header.colSpan">
              <FlexRender
                v-if="!header.isPlaceholder"
                :render="header.column.columnDef.header"
                :props="header.getContext()"
              />
            </TableHead>
          </TableRow>
        </TableHeader>
        <TableBody>
          <template v-if="loading">
            <TableRow v-for="i in 5" :key="i">
              <TableCell v-for="j in columns.length" :key="j" class="py-4">
                <div class="h-4 w-20 animate-pulse rounded bg-muted" />
              </TableCell>
            </TableRow>
          </template>
          <template v-else-if="table.getRowModel().rows.length">
            <TableRow
              v-for="row in table.getRowModel().rows"
              :key="row.id"
              :class="rowClassName?.(row.original)"
              @click="onRowClick?.(row.original)"
            >
              <TableCell v-for="cell in row.getVisibleCells()" :key="cell.id">
                <FlexRender :render="cell.column.columnDef.cell" :props="cell.getContext()" />
              </TableCell>
            </TableRow>
          </template>
          <TableRow v-else>
            <TableCell :colspan="columns.length" class="h-24 text-center">
              {{ emptyText }}
            </TableCell>
          </TableRow>
        </TableBody>
      </Table>
    </div>
    <Pagination
      v-if="pagination"
      :page="pagination.page"
      :page-size="pagination.pageSize"
      :total="pagination.total"
      :page-size-options="pagination.pageSizeOptions"
      @page-change="pagination.onPageChange"
      @page-size-change="pagination.onPageSizeChange"
    />
  </div>
</template>

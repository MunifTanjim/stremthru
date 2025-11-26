import {
  ColumnDef,
  getCoreRowModel,
  TableMeta,
  TableOptions,
} from "@tanstack/react-table";
import { useReactTable } from "@tanstack/react-table";

const coreRowModelGetter = getCoreRowModel();

export function useDataTable<TData, TValue>(
  options: Omit<TableOptions<TData>, "columns" | "getCoreRowModel" | "meta"> & {
    columns: ColumnDef<TData, TValue>[];
    meta?: TableMeta<TData>;
  },
) {
  const table = useReactTable({
    ...options,
    getCoreRowModel: coreRowModelGetter,
  });
  return table;
}

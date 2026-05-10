import { Cell, Header, RowData, Table as TTable } from "@tanstack/react-table";
import { flexRender } from "@tanstack/react-table";
import { CSSProperties } from "react";

import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import { cn } from "@/lib/utils";

export interface DataTableMetaCtx {
  "": void;
}

export interface DataTableMetaCtxKey {
  "": string;
}

declare module "@tanstack/react-table" {
  interface TableMeta<TData extends RowData> {
    ctx: DataTableMetaCtx[{
      [K in keyof DataTableMetaCtxKey]: TData extends DataTableMetaCtxKey[K]
        ? K
        : never;
    }[keyof DataTableMetaCtxKey]];
  }
}

export function DataTable<TData>({ table }: { table: TTable<TData> }) {
  return (
    <div className="overflow-hidden rounded-md border">
      <Table className="border-separate">
        <TableHeader>
          {table.getHeaderGroups().map((headerGroup) => (
            <TableRow key={headerGroup.id}>
              {headerGroup.headers.map((header) => {
                const pinned = header.column.getIsPinned();
                const style = prepareCellStyle({ cell: header });
                return (
                  <TableHead
                    className={cn("border-b", {
                      "bg-background sticky z-20": Boolean(pinned),
                      "border-l-1": pinned === "right",
                      "border-r-1": pinned === "left",
                    })}
                    key={header.id}
                    style={style}
                  >
                    {header.isPlaceholder
                      ? null
                      : flexRender(
                          header.column.columnDef.header,
                          header.getContext(),
                        )}
                  </TableHead>
                );
              })}
            </TableRow>
          ))}
        </TableHeader>
        <TableBody>
          {table.getRowModel().rows?.length ? (
            table.getRowModel().rows.map((row) => (
              <TableRow
                className="[&:last-child_td]:border-b-0"
                data-state={row.getIsSelected() && "selected"}
                key={row.id}
              >
                {row.getVisibleCells().map((cell) => {
                  const pinned = cell.column.getIsPinned();
                  const style = prepareCellStyle({ cell });
                  return (
                    <TableCell
                      className={cn("border-b", {
                        "bg-background sticky z-20": Boolean(pinned),
                        "border-l-1": pinned === "right",
                        "border-r-1": pinned === "left",
                      })}
                      key={cell.id}
                      style={style}
                    >
                      {flexRender(
                        cell.column.columnDef.cell,
                        cell.getContext(),
                      )}
                    </TableCell>
                  );
                })}
              </TableRow>
            ))
          ) : (
            <TableRow>
              <TableCell
                className="h-24 text-center"
                colSpan={table.options.columns.length}
              >
                No results.
              </TableCell>
            </TableRow>
          )}
        </TableBody>
      </Table>
    </div>
  );
}

function prepareCellStyle<TData>({
  cell,
}: {
  cell: Cell<TData, unknown> | Header<TData, unknown>;
}): CSSProperties | undefined {
  const style: CSSProperties = {};
  if (cell.column.columnDef.size) {
    style.width = `${cell.column.getSize()}px`;
    style.minWidth = style.width;
  }

  const pinPosition = cell.column.getIsPinned();
  if (pinPosition === "left") {
    style.left = cell.column.getStart("left");
  } else if (pinPosition === "right") {
    style.right = cell.column.getAfter("right");
  }

  return style;
}

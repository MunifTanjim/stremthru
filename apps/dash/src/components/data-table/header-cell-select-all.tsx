import { HeaderContext } from "@tanstack/react-table";

import { Checkbox } from "@/components/ui/checkbox";

export function HeaderCellSelectAll<TData, TValue>({
  table,
}: HeaderContext<TData, TValue>) {
  return (
    <Checkbox
      aria-label="Select all"
      checked={
        table.getIsAllPageRowsSelected() ||
        (table.getIsSomePageRowsSelected() && "indeterminate")
      }
      onCheckedChange={(value) => table.toggleAllPageRowsSelected(!!value)}
    />
  );
}

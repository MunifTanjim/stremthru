import { CellContext } from "@tanstack/react-table";

import { Checkbox } from "@/components/ui/checkbox";

export function CellSelect<TData, TValue>({ row }: CellContext<TData, TValue>) {
  return (
    <Checkbox
      aria-label="Select row"
      checked={row.getIsSelected()}
      onCheckedChange={(value) => row.toggleSelected(!!value)}
    />
  );
}

import { CommandLoading } from "cmdk";
import { SearchIcon } from "lucide-react";
import { useState } from "react";

import { AniDBTitle, useAniDBAutocomplete } from "@/api/anidb";
import { useDebouncedValue } from "@/hooks/use-debounced-value";

import { Button } from "./ui/button";
import {
  Command,
  CommandEmpty,
  CommandInput,
  CommandItem,
  CommandList,
} from "./ui/command";
import {
  Item,
  ItemContent,
  ItemDescription,
  ItemHeader,
  ItemTitle,
} from "./ui/item";
import { Popover, PopoverContent, PopoverTrigger } from "./ui/popover";

export function AniDBSearch({
  onSelect,
  triggerLabel = "Search...",
}: {
  onSelect: (title: AniDBTitle) => void;
  triggerLabel?: string;
}) {
  const [searchOpen, setSearchOpen] = useState(false);
  const [_searchQuery, setSearchQuery] = useState("");
  const searchQuery = useDebouncedValue(_searchQuery, 300);
  const autocompleteResults = useAniDBAutocomplete(searchQuery);

  return (
    <Popover onOpenChange={setSearchOpen} open={searchOpen}>
      <PopoverTrigger asChild>
        <Button
          aria-expanded={searchOpen}
          className="w-full justify-between"
          role="combobox"
          variant="outline"
        >
          <span className="truncate">{triggerLabel}</span>
          <SearchIcon className="opacity-50" />
        </Button>
      </PopoverTrigger>
      <PopoverContent className="w-[var(--radix-popover-trigger-width)] p-0">
        <Command shouldFilter={false}>
          <CommandInput
            onValueChange={setSearchQuery}
            placeholder="Search AniDB titles..."
            value={_searchQuery}
          />
          <CommandList>
            {autocompleteResults.isLoading && _searchQuery ? (
              <CommandLoading className="py-6 text-center text-sm">
                Searching...
              </CommandLoading>
            ) : (
              <CommandEmpty>AniDB Titles</CommandEmpty>
            )}
            {autocompleteResults.data?.map((title) => (
              <CommandItem
                key={title.id}
                onSelect={async () => {
                  onSelect(title);
                  setSearchQuery("");
                  setSearchOpen(false);
                }}
                value={title.id}
              >
                <Item className="w-full p-0" size="sm">
                  <ItemHeader className="text-muted-foreground flex justify-between text-xs">
                    <div>{title.type}</div>
                    <div>{title.id}</div>
                  </ItemHeader>
                  <ItemContent>
                    <ItemTitle>{title.title}</ItemTitle>
                    <ItemDescription>
                      <span className="text-muted-foreground text-xs">
                        {title.season && `S${title.season}`}
                        {title.season && title.year && " · "}
                        {title.year}
                      </span>
                    </ItemDescription>
                  </ItemContent>
                </Item>
              </CommandItem>
            ))}
          </CommandList>
        </Command>
      </PopoverContent>
    </Popover>
  );
}

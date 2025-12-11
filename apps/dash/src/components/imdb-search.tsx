import { SearchIcon } from "lucide-react";
import { useState } from "react";

import { IMDBTitle, useIMDBAutocomplete } from "@/api/imdb";
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

export function IMDBSearch({
  onSelect,
}: {
  onSelect: (title: IMDBTitle) => void;
}) {
  const [searchOpen, setSearchOpen] = useState(false);
  const [_searchQuery, setSearchQuery] = useState("");
  const searchQuery = useDebouncedValue(_searchQuery, 300);
  const autocompleteResults = useIMDBAutocomplete(searchQuery);

  return (
    <Popover onOpenChange={setSearchOpen} open={searchOpen}>
      <PopoverTrigger asChild>
        <Button
          aria-expanded={searchOpen}
          className="w-full justify-between"
          role="combobox"
          variant="outline"
        >
          Search...
          <SearchIcon className="opacity-50" />
        </Button>
      </PopoverTrigger>
      <PopoverContent className="w-[var(--radix-popover-trigger-width)] p-0">
        <Command shouldFilter={false}>
          <CommandInput
            onValueChange={setSearchQuery}
            placeholder="Search IMDB titles..."
            value={_searchQuery}
          />
          <CommandList>
            <CommandEmpty>IMDB Titles</CommandEmpty>
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

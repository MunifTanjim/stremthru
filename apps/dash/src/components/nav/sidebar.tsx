import { DropdownMenuGroup } from "@radix-ui/react-dropdown-menu";
import { Link, useMatchRoute } from "@tanstack/react-router";
import { capitalize } from "es-toolkit";
import {
  CalendarSyncIcon,
  ChevronRight,
  ChevronsUpDown,
  LayoutList,
  Lock,
  LogOut,
  MagnetIcon,
  Moon,
  Sparkles,
  Sun,
  User,
} from "lucide-react";
import { LayoutDashboard, type LucideIcon } from "lucide-react";
import { ComponentProps, useMemo } from "react";

import { useSignOut } from "@/api/auth";
import { useServerStats } from "@/api/stats";
import { Avatar, AvatarFallback } from "@/components/ui/avatar";
import {
  Collapsible,
  CollapsibleContent,
  CollapsibleTrigger,
} from "@/components/ui/collapsible";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import {
  Sidebar,
  SidebarContent,
  SidebarFooter,
  SidebarGroup,
  SidebarGroupLabel,
  SidebarHeader,
  SidebarMenu,
  SidebarMenuButton,
  SidebarMenuItem,
  SidebarMenuSub,
  SidebarMenuSubButton,
  SidebarMenuSubItem,
  SidebarRail,
  useSidebar,
} from "@/components/ui/sidebar";
import { useCurrentUser } from "@/hooks/auth";
import { FileRouteTypes } from "@/routeTree.gen";

import { useTheme } from "../theme";

type NavItem = {
  icon?: LucideIcon;
  items?: Pick<NavItem, "path" | "title">[];
  path: FileRouteTypes["to"];
  title: string;
};

export function DashSidebar({ ...props }: ComponentProps<typeof Sidebar>) {
  const serverStats = useServerStats();

  return (
    <Sidebar collapsible="icon" {...props}>
      <SidebarHeader>
        <SidebarMenu>
          <SidebarMenuItem>
            <SidebarMenuButton asChild size="lg">
              <a href="#">
                <div className="bg-sidebar-primary text-sidebar-primary-foreground flex aspect-square size-8 items-center justify-center rounded-lg">
                  <Sparkles className="size-4" />
                </div>
                <div className="flex flex-col gap-0.5 leading-none">
                  <span className="font-medium">StremThru</span>
                  <span className="">v{serverStats.data?.version}</span>
                </div>
              </a>
            </SidebarMenuButton>
          </SidebarMenuItem>
        </SidebarMenu>
      </SidebarHeader>
      <SidebarContent>
        <NavGroup />
      </SidebarContent>
      <SidebarFooter>
        <NavUser />
      </SidebarFooter>
      <SidebarRail />
    </Sidebar>
  );
}

function NavGroup() {
  const matchRoute = useMatchRoute();
  const navItems = useNavItems();

  return (
    <SidebarGroup>
      <SidebarGroupLabel>Platform</SidebarGroupLabel>
      <SidebarMenu>
        {navItems.map((item) => (
          <Collapsible
            asChild
            className="group/collapsible"
            defaultOpen={true}
            key={item.title}
          >
            <SidebarMenuItem>
              <CollapsibleTrigger asChild>
                <SidebarMenuButton tooltip={item.title}>
                  {item.icon && <item.icon />}
                  <span>{item.title}</span>
                  {item.items ? (
                    <ChevronRight className="ml-auto transition-transform duration-200 group-data-[state=open]/collapsible:rotate-90" />
                  ) : null}
                </SidebarMenuButton>
              </CollapsibleTrigger>
              <CollapsibleContent>
                <SidebarMenuSub>
                  {item.items?.map((subItem) => {
                    const isActive = !!matchRoute({ to: subItem.path });
                    return (
                      <SidebarMenuSubItem key={subItem.title}>
                        <SidebarMenuSubButton asChild isActive={isActive}>
                          <Link to={subItem.path}>
                            <span>{subItem.title}</span>
                          </Link>
                        </SidebarMenuSubButton>
                      </SidebarMenuSubItem>
                    );
                  })}
                </SidebarMenuSub>
              </CollapsibleContent>
            </SidebarMenuItem>
          </Collapsible>
        ))}
      </SidebarMenu>
    </SidebarGroup>
  );
}

function NavUser() {
  const { isMobile } = useSidebar();
  const user = useCurrentUser();

  const signOut = useSignOut();

  const { setTheme, theme } = useTheme();

  return (
    <SidebarMenu>
      <SidebarMenuItem>
        <DropdownMenu>
          <DropdownMenuTrigger asChild>
            <SidebarMenuButton
              className="data-[state=open]:bg-sidebar-accent data-[state=open]:text-sidebar-accent-foreground"
              size="lg"
            >
              <Avatar className="h-8 w-8 rounded-lg">
                <AvatarFallback className="rounded-lg">
                  <User />
                </AvatarFallback>
              </Avatar>
              <div className="grid flex-1 text-left text-sm leading-tight">
                <span className="truncate font-medium">{user.id}</span>
              </div>
              <ChevronsUpDown className="ml-auto size-4" />
            </SidebarMenuButton>
          </DropdownMenuTrigger>
          <DropdownMenuContent
            align="end"
            className="w-(--radix-dropdown-menu-trigger-width) min-w-56 rounded-lg"
            side={isMobile ? "bottom" : "right"}
            sideOffset={4}
          >
            <DropdownMenuLabel className="p-0 font-normal">
              <div className="flex items-center gap-2 px-1 py-1.5 text-left text-sm">
                <Avatar className="h-8 w-8 rounded-lg">
                  <AvatarFallback className="rounded-lg">
                    <User />
                  </AvatarFallback>
                </Avatar>
                <div className="grid flex-1 text-left text-sm leading-tight">
                  <span className="truncate font-medium">{user.id}</span>
                </div>
              </div>
            </DropdownMenuLabel>
            <DropdownMenuSeparator />
            <DropdownMenuGroup>
              <DropdownMenuItem
                onSelect={(e) => {
                  e.preventDefault();
                  setTheme((theme) => {
                    switch (theme) {
                      case "dark":
                        return "system";
                      case "light":
                        return "dark";
                      case "system":
                        return "light";
                    }
                  });
                }}
              >
                <Sun className="h-[1.2rem] w-[1.2rem] rotate-0 scale-100 transition-all dark:-rotate-90 dark:scale-0" />
                <Moon className="absolute h-[1.2rem] w-[1.2rem] rotate-90 scale-0 transition-all dark:rotate-0 dark:scale-100" />
                <span className="sr-only">Toggle Theme</span>
                {capitalize(theme)}
              </DropdownMenuItem>
            </DropdownMenuGroup>
            <DropdownMenuSeparator />
            <DropdownMenuItem
              onSelect={async () => {
                await signOut.mutateAsync();
              }}
            >
              <LogOut />
              Log out
            </DropdownMenuItem>
          </DropdownMenuContent>
        </DropdownMenu>
      </SidebarMenuItem>
    </SidebarMenu>
  );
}

function useNavItems(): NavItem[] {
  const { data: server } = useServerStats();
  return useMemo(() => {
    const items: NavItem[] = [
      {
        icon: LayoutDashboard,
        items: [
          {
            path: "/dash",
            title: "Stats",
          },
          {
            path: "/dash/workers",
            title: "Workers",
          },
        ],
        path: "/dash",
        title: "Dashboard",
      },
      {
        icon: LayoutList,
        items: [
          {
            path: "/dash/lists",
            title: "Stats",
          },
        ],
        path: "/dash/lists",
        title: "Lists",
      },
    ];

    const torrents: NavItem = {
      icon: MagnetIcon,
      items: [
        {
          path: "/dash/torrents",
          title: "Stats",
        },
      ],
      path: "/dash/torrents",
      title: "Torrents",
    };
    if (server?.feature.vault) {
      torrents.items!.push({
        path: "/dash/torrents/indexers-sync",
        title: "Indexers Sync",
      });
    }
    items.push(torrents);

    if (server?.feature.vault) {
      const vault: NavItem = {
        icon: Lock,
        items: [
          {
            path: "/dash/vault",
            title: "Overview",
          },
        ],
        path: "/dash/vault",
        title: "Vault",
      };
      vault.items!.push({
        path: "/dash/vault/stremio-accounts",
        title: "Stremio Accounts",
      });
      if (server.integration.trakt) {
        vault.items!.push({
          path: "/dash/vault/trakt-accounts",
          title: "Trakt Accounts",
        });
      }
      vault.items!.push({
        path: "/dash/vault/torznab-indexers",
        title: "Torznab Indexers",
      });
      items.push(vault);

      const sync: NavItem = {
        icon: CalendarSyncIcon,
        items: [
          {
            path: "/dash/sync",
            title: "Overview",
          },
        ],
        path: "/dash/sync",
        title: "Sync",
      };
      sync.items!.push({
        path: "/dash/sync/stremio-stremio",
        title: "Stremio ↔ Stremio",
      });
      if (server.integration.trakt) {
        sync.items!.push({
          path: "/dash/sync/stremio-trakt",
          title: "Stremio ↔ Trakt",
        });
      }
      items.push(sync);
    }

    return items;
  }, [server?.feature.vault, server?.integration.trakt]);
}

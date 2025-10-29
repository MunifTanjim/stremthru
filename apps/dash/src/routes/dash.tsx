import {
  createFileRoute,
  Navigate,
  Outlet,
  useMatch,
} from "@tanstack/react-router";

import { DashBreadcrumb } from "@/components/nav/breadcrumb";
import { DashSidebar } from "@/components/nav/sidebar";
import { Separator } from "@/components/ui/separator";
import {
  SidebarInset,
  SidebarProvider,
  SidebarTrigger,
} from "@/components/ui/sidebar";
import { useCurrentAuth } from "@/hooks/auth";

export const Route = createFileRoute("/dash")({
  component: RouteComponent,
  staticData: {
    crumb: "Dashboard",
  },
});

function RouteComponent() {
  const { user } = useCurrentAuth();

  const loginRoute = useMatch({ from: "/dash/login", shouldThrow: false });
  if (!loginRoute && !user) {
    return <Navigate to="/dash/login" />;
  }

  if (!user) {
    return <Outlet />;
  }

  return (
    <SidebarProvider>
      <DashSidebar />
      <SidebarInset>
        <header className="group-has-data-[collapsible=icon]/sidebar-wrapper:h-12 flex h-16 shrink-0 items-center gap-2 transition-[width,height] ease-linear">
          <div className="flex items-center gap-2 px-4">
            <SidebarTrigger className="-ml-1" />
            <Separator
              className="mr-2 data-[orientation=vertical]:h-4"
              orientation="vertical"
            />
            <DashBreadcrumb />
          </div>
        </header>
        <div className="flex flex-1 flex-col gap-4 p-4 pt-0">
          <Outlet />
        </div>
      </SidebarInset>
    </SidebarProvider>
  );
}

import { isMatch, Link, useMatches } from "@tanstack/react-router";

import {
  Breadcrumb,
  BreadcrumbItem,
  BreadcrumbLink,
  BreadcrumbList,
  BreadcrumbPage,
  BreadcrumbSeparator,
} from "@/components/ui/breadcrumb";

export function DashBreadcrumb() {
  const matches = useMatches();

  const matchesWithCrumbs = matches.filter((match) =>
    isMatch(match, "staticData.crumb"),
  );
  const lastIndex = matchesWithCrumbs.length - 1;

  return (
    <Breadcrumb>
      <BreadcrumbList>
        {matchesWithCrumbs.map((crumb, index) => (
          <div className="contents" key={crumb.fullPath}>
            {index > 0 && <BreadcrumbSeparator className="hidden md:block" />}
            <BreadcrumbItem className={index === 0 ? "hidden md:block" : ""}>
              {index === lastIndex ? (
                <BreadcrumbPage>{crumb.staticData.crumb}</BreadcrumbPage>
              ) : (
                <BreadcrumbLink asChild>
                  <Link from={crumb.fullPath}>{crumb.staticData.crumb}</Link>
                </BreadcrumbLink>
              )}
            </BreadcrumbItem>
          </div>
        ))}
      </BreadcrumbList>
    </Breadcrumb>
  );
}

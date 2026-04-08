"use client";

import * as React from "react";
import Link from "next/link";
import { usePathname } from "next/navigation";

import { NavUser } from "@/components/nav-user";
import {
  Sidebar,
  SidebarContent,
  SidebarFooter,
  SidebarHeader,
  SidebarGroup,
  SidebarGroupLabel,
  SidebarGroupContent,
  SidebarMenu,
  SidebarMenuItem,
  SidebarMenuButton,
} from "@/components/ui/sidebar";
import {
  RiSlowDownLine,
  RiLeafLine,
  RiNavigationLine,
  RiCodeSSlashLine,
  RiGeminiLine,
  RiDatabase2Line,
} from "@remixicon/react";

// This is sample data.
const data = {
  user: {
    name: "Admin",
    email: "admin@hydradns",
    avatar: "",
  },
  navMain: [
    {
      title: "General",
      items: [
        {
          title: "Dashboard",
          url: "/dashboard",
          icon: RiSlowDownLine,
        },
        {
          title: "DNS Engine",
          url: "/dashboard/dns",
          icon: RiLeafLine,
        },
        {
          title: "Policies",
          url: "/dashboard/policies",
          icon: RiNavigationLine,
        },
        {
          title: "Blocklists",
          url: "/dashboard/blocklists",
          icon: RiDatabase2Line,
        },
        {
          title: "Logs",
          url: "/dashboard/logs",
          icon: RiGeminiLine,
        },
        {
          title: "Settings",
          url: "#",
          icon: RiCodeSSlashLine,
        },
      ],
    },
  ],
};

function SidebarLogo() {
  return (
    <div className="flex items-center gap-2 px-2 group-data-[collapsible=icon]:px-0 transition-[padding] duration-200 ease-in-out">
      <Link className="group/logo inline-flex items-center gap-2" href="/dashboard">
        <span className="sr-only">HydraDNS</span>
        <div className="flex items-center justify-center size-9 group-data-[collapsible=icon]:size-8 rounded-lg bg-primary text-primary-foreground font-bold text-sm transition-[width,height] duration-200 ease-in-out">
          H
        </div>
        <span className="font-semibold text-sm group-data-[collapsible=icon]:hidden transition-opacity duration-200">
          HydraDNS
        </span>
      </Link>
    </div>
  );
}

export function AppSidebar({ ...props }: React.ComponentProps<typeof Sidebar>) {
  const pathname = usePathname();
  return (
    <Sidebar collapsible="icon" variant="inset" {...props}>
      <SidebarHeader className="h-16 max-md:mt-2 mb-2 justify-center">
        <SidebarLogo />
      </SidebarHeader>
      <SidebarContent className="-mt-2">
        {data.navMain.map((item) => (
          <SidebarGroup key={item.title}>
            <SidebarGroupLabel className="uppercase text-muted-foreground/65">
              {item.title}
            </SidebarGroupLabel>
            <SidebarGroupContent>
              <SidebarMenu>
                {item.items.map((item) => (
                  <SidebarMenuItem key={item.title}>
                    <SidebarMenuButton
                      asChild
                      className="group/menu-button group-data-[collapsible=icon]:px-[5px]! font-medium gap-3 h-9 [&>svg]:size-auto"
                      tooltip={item.title}
                      isActive={item.url === "/dashboard" ? pathname === "/dashboard" : pathname.startsWith(item.url)}
                    >
                      <a href={item.url}>
                        {item.icon && (
                          <item.icon
                            className="text-muted-foreground/65 group-data-[active=true]/menu-button:text-primary"
                            size={22}
                            aria-hidden="true"
                          />
                        )}
                        <span>{item.title}</span>
                      </a>
                    </SidebarMenuButton>
                  </SidebarMenuItem>
                ))}
              </SidebarMenu>
            </SidebarGroupContent>
          </SidebarGroup>
        ))}
      </SidebarContent>
      <SidebarFooter>
        <NavUser user={data.user} />
      </SidebarFooter>
    </Sidebar>
  );
}

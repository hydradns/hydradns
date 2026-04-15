"use client";

import * as React from "react";
import Link from "next/link";
import { usePathname } from "next/navigation";
import Image from "next/image";
import {
  LayoutDashboard,
  Shield,
  ShieldBan,
  ScrollText,
  ListOrdered,
  Settings,
} from "lucide-react";

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
          icon: LayoutDashboard,
        },
        {
          title: "DNS Engine",
          url: "/dashboard/dns",
          icon: Shield,
        },
        {
          title: "Blocklists",
          url: "/dashboard/blocklists",
          icon: ShieldBan,
        },
        {
          title: "Policies",
          url: "/dashboard/policies",
          icon: ScrollText,
        },
        {
          title: "Query Logs",
          url: "/dashboard/logs",
          icon: ListOrdered,
        },
        {
          title: "Settings",
          url: "#",
          icon: Settings,
        },
      ],
    },
  ],
};

function SidebarLogo() {
  return (
    <div className="flex items-center gap-2 px-2 group-data-[collapsible=icon]:px-0 transition-[padding] duration-200 ease-in-out">
      <Link
        className="group/logo inline-flex items-center gap-2"
        href="/dashboard"
      >
        <span className="sr-only">HydraDNS</span>
        <Image
          src="/logo.png"
          alt="HydraDNS"
          width={36}
          height={36}
          className="size-9 group-data-[collapsible=icon]:size-8 rounded-lg transition-[width,height] duration-200 ease-in-out"
        />
        <div className="flex flex-col group-data-[collapsible=icon]:hidden transition-opacity duration-200">
          <span className="font-semibold text-sm text-[#00D4AA]">
            HydraDNS
          </span>
          <span className="text-[10px] leading-tight text-muted-foreground/50">
            Network Protected
          </span>
        </div>
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
        {data.navMain.map((group) => (
          <SidebarGroup key={group.title}>
            <SidebarGroupLabel className="uppercase text-muted-foreground/65">
              {group.title}
            </SidebarGroupLabel>
            <SidebarGroupContent>
              <SidebarMenu>
                {group.items.map((item) => (
                  <SidebarMenuItem key={item.title}>
                    <SidebarMenuButton
                      asChild
                      className="group/menu-button group-data-[collapsible=icon]:px-[5px]! font-medium gap-3 h-9 [&>svg]:size-auto"
                      tooltip={item.title}
                      isActive={
                        item.url === "/dashboard"
                          ? pathname === "/dashboard"
                          : pathname.startsWith(item.url)
                      }
                    >
                      <a href={item.url}>
                        {item.icon && (
                          <item.icon
                            className="text-muted-foreground/65 group-data-[active=true]/menu-button:text-[#00D4AA]"
                            size={20}
                            aria-hidden="true"
                          />
                        )}
                        <span className="group-data-[active=true]/menu-button:text-[#00D4AA]">
                          {item.title}
                        </span>
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

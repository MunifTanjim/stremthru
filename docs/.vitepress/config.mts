import { defineConfig } from "vitepress";

export default defineConfig({
  cleanUrls: true,

  title: "StremThru",
  description: "Companion for Stremio",

  head: [
    [
      "link",
      {
        rel: "icon",
        href: "https://emojiapi.dev/api/v1/sparkles/256.png",
      },
    ],
  ],

  themeConfig: {
    nav: [
      { text: "Guide", link: "/getting-started/introduction" },
      { text: "Configuration", link: "/configuration/" },
      { text: "API", link: "/api/" },
    ],

    sidebar: [
      {
        text: "Getting Started",
        items: [
          {
            text: "Introduction",
            link: "/getting-started/introduction",
          },
          {
            text: "Installation",
            link: "/getting-started/installation",
          },
          { text: "Quick Start", link: "/getting-started/quick-start" },
        ],
      },
      {
        text: "Configuration",
        items: [
          { text: "Overview", link: "/configuration/" },
          {
            text: "Environment Variables",
            link: "/configuration/environment-variables",
          },
          { text: "Features", link: "/configuration/features" },
          { text: "Usenet", link: "/configuration/usenet" },
          { text: "Database & Redis", link: "/configuration/database" },
        ],
      },
      {
        text: "Stremio Addons",
        items: [
          { text: "Overview", link: "/stremio-addons/" },
          { text: "Store", link: "/stremio-addons/store" },
          { text: "Wrap", link: "/stremio-addons/wrap" },
          { text: "Sidekick", link: "/stremio-addons/sidekick" },
          { text: "Torz", link: "/stremio-addons/torz" },
          { text: "Newz", link: "/stremio-addons/newz" },
          { text: "List", link: "/stremio-addons/list" },
        ],
      },
      {
        text: "API",
        items: [
          { text: "Overview", link: "/api/" },
          { text: "Proxy", link: "/api/proxy" },
          { text: "Store", link: "/api/store" },
          { text: "Newz", link: "/api/newz" },
          { text: "Meta", link: "/api/meta" },
        ],
      },
      {
        text: "Integrations",
        items: [
          { text: "Overview", link: "/integrations/" },
          { text: "TMDB", link: "/integrations/tmdb" },
          { text: "Trakt", link: "/integrations/trakt" },
          { text: "AniList", link: "/integrations/anilist" },
          { text: "Letterboxd", link: "/integrations/letterboxd" },
          { text: "MDBList", link: "/integrations/mdblist" },
          { text: "TVDB", link: "/integrations/tvdb" },
          { text: "GitHub", link: "/integrations/github" },
        ],
      },
      {
        text: "SDK",
        items: [
          { text: "Overview", link: "/sdk/" },
          { text: "JavaScript", link: "/sdk/javascript" },
          { text: "Python", link: "/sdk/python" },
        ],
      },
      {
        text: "Deployment",
        items: [
          { text: "Overview", link: "/deployment/" },
          { text: "Docker", link: "/deployment/docker" },
          { text: "Reverse Proxy", link: "/deployment/reverse-proxy" },
          {
            text: "Cloudflare WARP",
            link: "/deployment/cloudflare-warp",
          },
        ],
      },
    ],

    socialLinks: [
      {
        icon: "github",
        link: "https://github.com/MunifTanjim/stremthru",
      },
    ],

    editLink: {
      pattern: "https://github.com/MunifTanjim/stremthru/edit/main/docs/:path",
    },

    search: {
      provider: "local",
    },
  },
});

import { defineConfig } from "vitepress";

export default defineConfig({
  cleanUrls: true,
  lastUpdated: true,

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

  sitemap: {
    hostname: "https://docs.stremthru.13377001.xyz",
  },

  themeConfig: {
    nav: [
      { text: "Guide", link: "/guides/docker-compose" },
      { text: "Configuration", link: "/configuration/" },
      { text: "API", link: "/api/" },
    ],

    sidebar: [
      {
        text: "Getting Started",
        items: [
          { text: "Introduction", link: "/getting-started/introduction" },
          { text: "Installation", link: "/getting-started/installation" },
          { text: "FAQ", link: "/getting-started/faq" },
        ],
      },
      {
        text: "Guides",
        items: [
          { text: "Docker Compose", link: "/guides/docker-compose" },
          { text: "Stream Filter", link: "/guides/stream-filter" },
          { text: "Usenet Setup", link: "/guides/usenet-setup" },
        ],
      },
      {
        text: "Configuration",
        items: [
          { text: "Overview", link: "/configuration/" },
          {
            text: "Database & Cache",
            link: "/configuration/database-and-cache",
          },
          { text: "Features", link: "/configuration/features" },
          { text: "Integrations", link: "/configuration/integrations" },
          { text: "Stremio Addons", link: "/configuration/stremio-addons" },
          { text: "Newz", link: "/configuration/newz" },
          { text: "Torz", link: "/configuration/torz" },
        ],
      },
      {
        text: "Stremio Addons",
        items: [
          { text: "Overview", link: "/stremio-addons/" },
          { text: "List", link: "/stremio-addons/list" },
          { text: "Store", link: "/stremio-addons/store" },
          { text: "Wrap", link: "/stremio-addons/wrap" },
          { text: "Torz", link: "/stremio-addons/torz" },
          { text: "Newz", link: "/stremio-addons/newz" },
          { text: "Sidekick", link: "/stremio-addons/sidekick" },
        ],
      },
      {
        text: "API",
        items: [
          { text: "Overview", link: "/api/" },
          { text: "Proxy", link: "/api/proxy" },
          { text: "Store", link: "/api/store" },
          { text: "Newz", link: "/api/newz" },
          { text: "Torz", link: "/api/torz" },
          { text: "Meta", link: "/api/meta" },
        ],
      },
      {
        text: "Integrations",
        items: [
          { text: "Overview", link: "/integrations/" },
          { text: "AniList", link: "/integrations/anilist" },
          { text: "GitHub", link: "/integrations/github" },
          { text: "Letterboxd", link: "/integrations/letterboxd" },
          { text: "MDBList", link: "/integrations/mdblist" },
          { text: "Serializd", link: "/integrations/serializd" },
          { text: "TMDB", link: "/integrations/tmdb" },
          { text: "TVDB", link: "/integrations/tvdb" },
          { text: "Trakt", link: "/integrations/trakt" },
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
    ],

    socialLinks: [
      {
        icon: "github",
        link: "https://github.com/MunifTanjim/stremthru",
      },
      {
        icon: "discord",
        link: "https://go.muniftanjim.dev/discord",
      },
      {
        icon: "buymeacoffee",
        link: "https://buymeacoffee.com/muniftanjim",
      },
      {
        icon: "patreon",
        link: "https://www.patreon.com/muniftanjim",
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

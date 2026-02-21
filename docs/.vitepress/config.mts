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
          { text: "Introduction", link: "/getting-started/introduction" },
          { text: "Installation", link: "/getting-started/installation" },
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
          {
            text: "Database & Cache",
            link: "/configuration/database-and-cache",
          },
          { text: "Stremio Addons", link: "/configuration/stremio-addons" },
          { text: "Integrations", link: "/configuration/integrations" },
          { text: "Features", link: "/configuration/features" },
          { text: "Newz", link: "/configuration/newz" },
          { text: "Torz", link: "/configuration/torz" },
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

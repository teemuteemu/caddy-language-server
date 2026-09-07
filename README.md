# caddy-language-server

A language server for [Caddyfile](https://caddyserver.com/docs/caddyfile) configuration files.

## Features

- **Diagnostics** — flags unknown directives, misplaced subdirectives, invalid subdirectives inside blocks, and undefined snippet references in `import` statements
- **Completion** — suggests top-level directives inside site blocks and snippet names after `import`
- **Hover** — shows documentation for directives under the cursor

The parser is built on Caddy's own tokenizer (`github.com/caddyserver/caddy/v2/caddyconfig/caddyfile`) so it stays in sync with Caddy's actual syntax rules.

## Install

```
go install github.com/teemuteemu/caddy-language-server@latest
```

This puts a `caddy-language-server` binary in `$(go env GOPATH)/bin` — make sure that directory is on your `PATH`. Prebuilt binaries for Linux and macOS are also attached to each [release](https://github.com/teemuteemu/caddy-language-server/releases).

## Editor setup

caddy-language-server communicates over stdio using the Language Server Protocol (JSON-RPC 2.0). Point your editor's LSP client at the `caddy-language-server` binary with no extra arguments.

**Neovim (nvim-lspconfig)**

```lua
vim.api.nvim_create_autocmd({ "BufRead", "BufNewFile" }, {
  pattern = "Caddyfile",
  callback = function()
    vim.bo.filetype = "caddy"
  end,
})

require("lspconfig.configs").caddy_ls = {
  default_config = {
    cmd = { "caddy-language-server" },
    filetypes = { "caddy" },
    root_dir = require("lspconfig.util").root_pattern("Caddyfile"),
  },
}
require("lspconfig").caddy_ls.setup({})
```

---

> [!NOTE]
> **Neovim 0.11+**
>
> The configuration API changed in Neovim 0.11. The `require("lspconfig").setup()` API is now deprecated in favor of the native `vim.lsp.config()` and `vim.lsp.enable()` APIs. The configuration above continues to work with older versions of Neovim.
>
> For Neovim 0.11+, use:
>
> ```lua
> vim.filetype.add({
>   filename = {
>     Caddyfile = "caddy",
>   },
> })
>
> vim.lsp.config("caddy_ls", {
>   cmd = { "caddy-language-server" },
>   filetypes = { "caddy" },
>   root_markers = { "Caddyfile" },
> })
>
> vim.lsp.enable("caddy_ls")
> ```
>
> This requires **Neovim 0.11 or newer**. See the [nvim-lspconfig migration instructions](https://github.com/neovim/nvim-lspconfig#migration-instructions) for more information.
> 



## Development

```
go test ./...   # run tests
go vet ./...    # static analysis
```

## License

MIT

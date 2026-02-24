# caddy-ls

A language server for [Caddyfile](https://caddyserver.com/docs/caddyfile) configuration files.

## Features

- **Diagnostics** — flags unknown directives, misplaced subdirectives, invalid subdirectives inside blocks, and undefined snippet references in `import` statements
- **Completion** — suggests top-level directives inside site blocks and snippet names after `import`
- **Hover** — shows documentation for directives under the cursor

The parser is built on Caddy's own tokenizer (`github.com/caddyserver/caddy/v2/caddyconfig/caddyfile`) so it stays in sync with Caddy's actual syntax rules.

## Editor setup

caddy-ls communicates over stdio using the Language Server Protocol (JSON-RPC 2.0). Point your editor's LSP client at the `caddy-ls` binary with no extra arguments.

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
    cmd = { "caddy-ls" },
    filetypes = { "caddy" },
    root_dir = require("lspconfig.util").root_pattern("Caddyfile"),
  },
}
require("lspconfig").caddy_ls.setup({})
```

## Development

```
go test ./...   # run tests
go vet ./...    # static analysis
```

## License

MIT

// Language registry for listings.typ backed by the languages compiled into inkjet.wasm.
// The returned value is the token passed to the Inkjet Typst plugin.

#let listings-language-aliases = (
  "rust": "rust",
  "rs": "rust",
  "python": "python",
  "py": "python",
  "python3": "python",
  "javascript": "javascript",
  "js": "javascript",
  "jsx": "javascript",
  "typescript": "typescript",
  "ts": "typescript",
  "tsx": "typescript",
  "json": "json",
  "jsonc": "json",
  "bash": "bash",
  "sh": "bash",
  "shell": "bash",
  "zsh": "bash",
  "c": "c",
  "h": "c",
  "cpp": "cpp",
  "c++": "cpp",
  "cc": "cpp",
  "cxx": "cpp",
  "hpp": "cpp",
  "go": "go",
  "golang": "go",
  "java": "java",
  "toml": "toml",
  "css": "css",
  "html": "html",
  "htm": "html",
  "lua": "lua",
)

// Names commonly used by listings-compatible callers. These are retained as
// metadata for callers that inspect the registry.
#let listings-driver-names = (
  "rust": "rust",
  "python": "python",
  "javascript": "javascript",
  "typescript": "typescript",
  "json": "json",
  "bash": "bash",
  "c": "c",
  "cpp": "cpp",
  "go": "go",
  "java": "java",
  "toml": "toml",
  "css": "css",
  "html": "html",
  "lua": "lua",
)

#let listings-language-registry(extra: (:)) = {
  listings-language-aliases + extra
}

#let listings-add-language(registry: listings-language-aliases, name, token: none, aliases: ()) = {
  let canonical = if token == none { name } else { token }
  let result = registry + (str(name): canonical)
  for alias in aliases {
    result = result + (str(alias): canonical)
  }
  result
}

#let listings-language(name, registry: listings-language-aliases) = {
  if name == none { none }
  else {
    let key = lower(str(name))
    registry.at(key, default: none)
  }
}

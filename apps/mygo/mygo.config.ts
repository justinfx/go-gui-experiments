// Plain object: mygo-cli lives in frontend/node_modules, which the config at the project root cannot resolve.
// import { defineConfig } from "mygo-cli";

export default ({
  name: "gguix-mygo",
  identifier: "dev.gguix.mygo",
  version: "0.0.0",
  main: ".",
  devUrl: "http://127.0.0.1:9245",
  devCommand: "npm --prefix frontend run dev",
  buildCommand: "npm --prefix frontend run build",
  frontendDist: "frontend/dist",
  bindings: "frontend/src/mygo.ts",
  out: "build/out",
});

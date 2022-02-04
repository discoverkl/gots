import { getapi, Base } from "vue2go";

const dev = false || true;
let api: API;

interface API extends Base {
  listDir(path: string): FileInfo[];
  loadText(path: string): string;
  saveText(path: string, text: string);
}

interface FileInfo {
  name: string;
  path: string;
  isDir?: boolean;
}

try {
  api = getapi() as API;
} catch (ex) {
  // provide mock API in dev mode
  if (!dev) console.error(`API is not ready: ${ex}`);
  else
    api = {
      loadText(path) {
        return `text content of file: ${path}`;
      },
      saveText(path, text) {},
      listDir(path) {
        switch (path) {
          case "":
            return [
              {
                name: "dist",
                path: "dist",
                isDir: true
              },
              {
                name: "src",
                path: "src",
                isDir: true
              },
              {
                name: "package.json",
                path: "package.json"
              }
            ];
          case "dist":
            return [
              {
                name: "js",
                path: "dist/js",
                isDir: true
              },
              {
                name: "index.html",
                path: "dist/index.html"
              }
            ];
          case "src":
            return [
              {
                name: "main.ts",
                path: "src/main.ts"
              },
              {
                name: "App.vue",
                path: "src/App.vue"
              }
            ];
          case "dist/js":
            return [
              {
                name: "app.js",
                path: "dist/js/app.ts"
              }
            ];
          default:
            return [];
        }
      }
    } as API;
}

export default api;

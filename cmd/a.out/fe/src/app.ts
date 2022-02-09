interface Message {
  id?: number;
  method: string;
  params: any;
}

interface CallMessage {
  id?: number;
  method: string;
  params: {
    name: string;
    seq: number;
    args: any[];
  };
}

interface RefCallMessage {
  id?: number;
  method: string;
  params: {
    seq: number;
  };
}

interface Option {
  dev: boolean;
  tls: boolean;
  readyFuncName: string; // bind name for ready function
  prefix: string; // publicPath
  search: string; // search string used to fetch this script
  bindings: string[]; // server binding names
  blurOnClose: boolean; // make body blur on socket close
}

(function () {
  let options: Option;
  // inject server options here
  options = null;
  if (options === null) {
    options = {
      dev: true,
      tls: false,
      readyFuncName: "Gots",
      prefix: "",
      search: "?name=api",
      bindings: [],
      blurOnClose: true
    };
  }
  let dev = options.dev;
  class Gots {
    ws: WebSocket;
    root: any; // {}
    resolveAPI: any;
    lastRefID: number;
    contextType: any;
    beforeReady: () => void;

    constructor(ws: WebSocket) {
      this.ws = ws;
      this.resolveAPI = null;
      this.lastRefID = 0;
      this.beforeReady = null;

      this.buildRoot();
      this.attach();
      this.initContext();
    }

    buildRoot(): void {
      let root = {};
      const ready = new Promise((resolve, reject) => {
        this.resolveAPI = resolve;
      });
      // root[options.readyFuncName] = () => ready;
      root[options.readyFuncName] = {};
      for (const name of options.bindings) {
        let placeholder = async function () {
          await ready;
          if (root[name] === placeholder) {
            throw new Error("binding is not ready: " + name);
            return;
          }
          return await root[name](...arguments);
        };
        root[name] = placeholder;
        this.copyBind(name, root);
      }
      this.extendGots(root[options.readyFuncName]);
      this.root = root;
    }

    extendGots(Gots: any) {
      Gots.self = this;
    }

    getapi(): any {
      return this.root;
    }

    replymessage(id: number, ret?: any, err?: string) {
      if (ret === undefined) ret = null;
      if (err === undefined) err = null;
      let msg: Message = {
        id: id,
        method: "Gots.ret",
        params: {
          result: ret,
          error: err
        }
      };
      this.ws.send(JSON.stringify(msg));
    }

    onmessage(e: MessageEvent) {
      let ws = this.ws;
      let msg = JSON.parse(e.data);
      if (dev) console.log("receive: ", JSON.stringify(msg, null, "  "));
      let root = this.root;
      let method = msg.method;
      let params;
      switch (method) {
        case "Gots.call": {
          params = msg.params;
          switch (params.name) {
            case "eval": {
              let ret, err;
              try {
                ret = eval(params.args[0]);
              } catch (ex) {
                err = ex.toString() || "unknown error";
              }
              this.replymessage(msg.id, ret, err);
              break;
            }
          }
          break;
        }
        case "Gots.ret": {
          let { name, seq, result, error } = msg.params;
          if (error) {
            root[name]["errors"].get(seq)(error);
          } else {
            root[name]["results"].get(seq)(result);
          }
          root[name]["errors"].delete(seq);
          root[name]["results"].delete(seq);
          break;
        }
        case "Gots.callback": {
          let { name, seq, args } = msg.params;
          let ret, err;
          try {
            ret = root[name]["callbacks"].get(seq)(...args);
          } catch (ex) {
            err = ex.toString() || "unknown error";
          }
          this.replymessage(msg.id, ret, err);
          break;
        }
        case "Gots.closeCallback": {
          let { name, seq } = msg.params;
          root[name]["callbacks"].delete(seq);
          break;
        }
        case "Gots.bind": {
          params = msg.params;
          if (Array.isArray(params.name))
            for (const name of params.name) this.bind(name);
          else this.bind(params.name);
          break;
        }
        case "Gots.ready": {
          if (this.beforeReady !== null) {
            this.beforeReady();
          }
          if (this.resolveAPI != null) {
            this.resolveAPI();
          }
          break;
        }
      }
    }

    attach() {
      let ws = this.ws;
      ws.onmessage = this.onmessage.bind(this);

      ws.onopen = e => {
        if (options.blurOnClose)
          (window as any).document.body.style.opacity = 1;
      };

      ws.onerror = e => {
        console.log("ws error at", new Date().toLocaleString(), e);
      };

      ws.onclose = e => {
        if (options.blurOnClose)
          (window as any).document.body.style.opacity = 0.382;
        console.log("ws close at", new Date().toLocaleString(), e);
      };
    }

    bind(name: string) {
      let root = this.root;
      const bindingName = name;
      root[bindingName] = async (...args) => {
        const me = root[bindingName];

        for (let i = 0; i < args.length; i++) {
          // support javascript functions as arguments
          if (typeof args[i] == "function") {
            let callbacks = me["callbacks"];
            if (!callbacks) {
              callbacks = new Map();
              me["callbacks"] = callbacks;
            }
            const seq = (callbacks["lastSeq"] || 0) + 1;
            callbacks["lastSeq"] = seq;
            callbacks.set(seq, args[i]); // root[bindingName].callbacks[callbackSeq] = func value
            args[i] = {
              bindingName: bindingName,
              seq: seq
            };
          } else if (args[i] instanceof this.contextType) {
            const seq = ++this.lastRefID;
            // js: rewrite input Context().seq = seq
            args[i].seq = seq;
            // go: will create Context object from seq and put it in jsclient.refs
            args[i] = {
              seq: seq
            };
          }
        }

        // prepare (errors, results, lastSeq) on binding function
        let errors = me["errors"];
        let results = me["results"];
        if (!results) {
          results = new Map();
          me["results"] = results;
        }
        if (!errors) {
          errors = new Map();
          me["errors"] = errors;
        }
        const seq = (me["lastSeq"] || 0) + 1;
        me["lastSeq"] = seq;
        const promise = new Promise((resolve, reject) => {
          results.set(seq, resolve);
          errors.set(seq, reject);
        });

        // call go
        let callMsg: CallMessage = {
          method: "Gots.call",
          params: {
            name: bindingName,
            seq,
            args
          }
        };
        // binding call phrase 1
        this.ws.send(JSON.stringify(callMsg));
        return promise;
      };

      this.copyBind(bindingName, root);
    }

    copyBind(bindingName: string, root: {}) {
      // copy root["a.b"] to root.a.b
      if (bindingName.indexOf(".") !== -1) {
        const sp = bindingName.split(".");
        const [parts, name] = [sp.slice(0, sp.length - 1), sp[sp.length - 1]];
        let target = root;
        for (const part of parts) {
          target[part] = target[part] || {};
          target = target[part];
        }
        target[name] = root[bindingName];
      }
    }

    initContext() {
      let $this = this;

      // Context class
      function Context() {
        this.seq = -1; // this will be rewrite as refID
        this.cancel = () => {
          let msg: RefCallMessage = {
            method: "Gots.refCall",
            params: {
              seq: this.seq
            }
          };
          $this.ws.send(JSON.stringify(msg));
        };
        this.getThis = () => {
          return $this;
        };
      }
      this.contextType = Context;

      const TODO = new Context();
      const Backgroud = new Context();

      // context package
      this.root.context = {
        withCancel() {
          let ctx = new Context();
          return [ctx, ctx.cancel];
        },
        background() {
          return Backgroud;
        },
        todo() {
          return TODO;
        }
      };
    }
  }

  function getparam(name: string, search?: string): string | undefined {
    search = search === undefined ? window.location.search : search;
    let pair = search
      .slice(1)
      .split("&")
      .map(one => one.split("="))
      .filter(one => one[0] == name)
      .slice(-1)[0];
    if (pair === undefined) return;
    return pair[1] || "";
  }

  function main() {
    let host = window.location.host;
    let ws = new WebSocket(
      (options.tls ? "wss://" : "ws://") + host + options.prefix + "/gots"
    );
    let gots = new Gots(ws);
    let api = gots.getapi();

    let exportAPI = () => {
      let name = getparam("name", options.search);
      let win: any = window;
      if (name === undefined || name === "window") Object.assign(win, api);
      else if (name) win[name] = api;
    };
    gots.beforeReady = exportAPI;
    exportAPI();
    return api;
  }
  return main();
})();

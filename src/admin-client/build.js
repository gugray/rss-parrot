build();

const mainPort = 8090;

async function build() {

  const esbuild = require("esbuild");
  const { lessLoader } = require("esbuild-plugin-less");
  const livereload = require("livereload");
  const staticServer = require("static-server");
  const pubDir = "public";
  const watch = {
    onRebuild(error) {
      let dstr = "[" + new Date().toLocaleTimeString() + "] ";
      if (error) {
        console.error(dstr + "Change detected; rebuild failed:", error);
        return;
      }
      console.log(dstr + "Change detected; rebuild OK");
    },
  };

  esbuild.build({
    entryPoints: ["src/app.js", "src/app.less"],
    outdir: "public",
    bundle: true,
    sourcemap: true,
    minify: false,
    plugins: [
      lessLoader(),
    ],
    watch: watch,
  }).catch(err => {
    console.error("Unexpected error; quitting.");
    if (err) console.error(err);
    process.exit(1);
  }).then(() => {
    console.log("Build finished.");
    livereload.createServer().watch("./" + pubDir);
    console.log("Watching changes, with livereload...");
    const server = new staticServer({
      rootPath: "./" + pubDir,
      port: mainPort,
    });
    server.start(function () {
      console.log("Server listening at " + server.port + "; serving from " + pubDir);
    });
  });
}

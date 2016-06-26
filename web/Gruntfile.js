module.exports = function(grunt) {
    grunt.initConfig({
        pkg: grunt.file.readJSON("package.json"),

        clean: {
            folder: ["dist/"]
        },

        copy: {
            main: {
                files: [{
                    expand: true,
                    cwd: "src/css",
                    src: "**/*",
                    dest: "dist/css/"
                }, {
                    src: "src/index.html",
                    dest: "dist/index.html"
                }, {
                    expand: true,
                    cwd: "src/img",
                    src: "**/*",
                    dest: "dist/img/"
                }, ]
            }
        },
        browserify: {
            main: {
                src: "src/js/**.js",
                dest: "dist/js/bus.js"
            }
        },
    });

    grunt.loadNpmTasks("grunt-browserify");
    grunt.loadNpmTasks("grunt-contrib-copy");
    grunt.loadNpmTasks("grunt-contrib-clean");

    grunt.registerTask("default", ["clean", "copy", "browserify"]);
};

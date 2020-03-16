const gulp = require('gulp');
const sass = require('gulp-sass');
const autoprefixer = require('gulp-autoprefixer');
const minifyCSS = require("gulp-clean-css");
const rename = require("gulp-rename");

var paths = {
    style: {
        src: 'public/scss/style.scss',
        all: 'public/scss/**/**/*.scss',
        dest: 'public/css/'
    },
    style_alternative: {
        src: 'public/scss_alternative/style_alternative.scss',
        all: 'public/scss_alternative/**/**/*.scss',
        dest: 'public/css/'
    }
}

function style() {
    return gulp.src(paths.style.src)
    .pipe(sass().on('error', sass.logError))
    .pipe(autoprefixer({
        browsers: ['last 2 versions'],
        cascade: false
    }))
    .pipe(minifyCSS())
    .pipe(rename('style.min.css'))
    .pipe(gulp.dest(paths.style.dest));
}

function style_alternative() {
    return gulp.src(paths.style_alternative.src)
    .pipe(sass().on('error', sass.logError))
    .pipe(autoprefixer({
        browsers: ['last 2 versions'],
        cascade: false
    }))
    .pipe(minifyCSS())
    .pipe(rename('style_alternative.min.css'))
    .pipe(gulp.dest(paths.style_alternative.dest));
}

function watch() {
    gulp.watch(paths.style.all, style);
}

function watch_alternative() {
    gulp.watch(paths.style_alternative.all, style_alternative)
}

gulp.task('default', gulp.series(watch));
gulp.task('style', style);

gulp.task('watch-alternative', gulp.series(watch_alternative))
gulp.task('style-alternative', style_alternative)

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
    style_classic: {
        src: 'public/scss_classic/style_classic.scss',
        all: 'public/scss_classic/**/**/*.scss',
        dest: 'public/css/'
    },
    style_dark: {
        src: 'public/scss_dark/style_dark.scss',
        all: 'public/scss_dark/**/**/*.scss',
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

function style_classic() {
    return gulp.src(paths.style_classic.src)
    .pipe(sass().on('error', sass.logError))
    .pipe(autoprefixer({
        browsers: ['last 2 versions'],
        cascade: false
    }))
    .pipe(minifyCSS())
    .pipe(rename('style_classic.min.css'))
    .pipe(gulp.dest(paths.style_classic.dest));
}

function style_dark() {
    return gulp.src(paths.style_dark.src)
    .pipe(sass().on('error', sass.logError))
    .pipe(autoprefixer({
        browsers: ['last 2 versions'],
        cascade: false
    }))
    .pipe(minifyCSS())
    .pipe(rename('style_dark.min.css'))
    .pipe(gulp.dest(paths.style_dark.dest));
}

function watch() {
    gulp.watch(paths.style.all, style);
}

function watch_classic() {
    gulp.watch(paths.style_classic.all, style_classic)
}

function watch_dark() {
    gulp.watch(paths.style_dark.all, style_dark)
}

gulp.task('default', gulp.series(watch));
gulp.task('style', style);

gulp.task('watch-classic', gulp.series(watch_classic))
gulp.task('style-classic', style_classic)

gulp.task('watch-dark', gulp.series(watch_dark))
gulp.task('style-dark', style_dark)
const gulp = require('gulp');
const sass = require('gulp-sass');
const autoprefixer = require('gulp-autoprefixer');

var paths = {
    style: {
        src: 'src/static/scss/style.scss',
        all: 'src/static/scss/**/**/*.scss',
        dest: 'src/static/css/'
    }
}

function style() {
    return gulp.src(paths.style.src)
    .pipe(sass().on('error', sass.logError))
    .pipe(autoprefixer({
        browsers: ['last 2 versions'],
        cascade: false
    }))
    .pipe(gulp.dest(paths.style.dest));
}

function watch() {
    gulp.watch(paths.style.all, style);
}

gulp.task('default', gulp.series(watch));
gulp.task('style', style)

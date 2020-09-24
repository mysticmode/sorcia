'use strict';
 
var gulp = require('gulp');
var sass = require('gulp-sass');
var rename = require('gulp-rename');
 
sass.compiler = require('node-sass');

var paths = {
  style: {
    src: 'public/scss/style.scss',
    all: 'public/scss/**/**/*.scss',
    dest: 'public/css/',
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
 
gulp.task('sass', function () {
  return gulp.src(paths.style.src)
    .pipe(sass({outputStyle: 'compressed'}).on('error', sass.logError))
    .pipe(rename('style.min.css'))
    .pipe(gulp.dest(paths.style.dest));
});

gulp.task('sass_classic', function() {
return gulp.src(paths.style_classic.src)
    .pipe(sass({outputStyle: 'compressed'}).on('error', sass.logError))
    .pipe(rename('style_classic.min.css'))
    .pipe(gulp.dest(paths.style_classic.dest));
});

gulp.task('sass_dark', function() {
return gulp.src(paths.style_dark.src)
    .pipe(sass({outputStyle: 'compressed'}).on('error', sass.logError))
    .pipe(rename('style_dark.min.css'))
    .pipe(gulp.dest(paths.style_dark.dest));
});

 
gulp.task('default', gulp.series('sass'));

gulp.task('watch', () => {
  gulp.watch(paths.style.src, (done) => {
    gulp.series(['sass'])(done);
  });
});

gulp.task('default-classic', gulp.series('sass_classic'));

gulp.task('watch-classic', () => {
  gulp.watch(paths.style_classic.src, (done) => {
    gulp.series(['sass_classic'])(done);
  });
});

gulp.task('default-dark', gulp.series('sass_dark'));

gulp.task('watch-dark', () => {
  gulp.watch(paths.style_dark.src, (done) => {
    gulp.series(['sass_dark'])(done);
  });
});

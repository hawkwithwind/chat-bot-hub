// 引入 gulp
var gulp = require('gulp');
// 自定义组件工具
//var through = require('through2');
// 引入组件
var concat = require('gulp-concat');
var uglify = require('gulp-uglify');
var cssmin = require('gulp-cssmin');
var rename = require('gulp-rename');
var ngAnnotate = require('gulp-ng-annotate');
var html2js = require('gulp-html2js');
var revCollector = require('gulp-rev-collector');
var browserSync = require('browser-sync').create();

var concatConfig = {newLine: ';'};
var srcConfig = {cache: false, statCache: false, realpathCache: false};

var staticPath = "static/";
var libPath = "static/lib/";
var appPath = "static/app/";
var exculePath = "!static/app/";
var cssPath = "static/css/";
//var destPath = staticPath + 'dest/';
var destPath = '../build/static/';

// 库文件
var libConfig = [
  //'moment', 'local',
  libPath + 'moment/moment.min.js',
  libPath + 'moment/local/zh-cn.js',
  libPath + 'jquery/jquery-2.1.3.min.js',
  libPath + 'jquery/*.js',
  libPath + 'angular/angular.min.js',
  libPath + 'angular/*.js',
  libPath + 'bootstrap/js/bootstrap.min.js',
  //libPath + 'bootstrap-hover-dropdown/bootstrap-hover-dropdown.min.js',
  libPath + 'oclazyload/ocLazyLoad.min.js',
  libPath + 'loadingbar/loading-bar.min.js',
  libPath + 'angularstorage/ngStorage.min.js',
  libPath + 'angular-breadcrumb/angular-breadcrumb.min.js',
  libPath + 'angular-bootstrap-ui/ui-bootstrap-tpls-0.14.3.js',
  libPath + 'angular-scroll/angular-scroll.min.js',
  //libPath + 'angular-ui-grid/ui-grid.min.js',
  libPath + 'angular-ag-grid/ag-grid.min.js',
  //libPath + 'angular-ui-layout/ui-layout.js',
  libPath + 'bootstrap-bootbox/bootbox.min.js', // 弹出提示 
  libPath + 'jquery.blockui.min.js',			 // 遮罩层
  libPath + 'bootstrap-toastr/toastr.min.js',   // 消息提示
  libPath + 'highcharts-ng/*.js', //highcharts
  libPath + 'tablecsv/tablecsv.js'
];

// 启动js文件
var appConfig = [
  // 加载服务
  appPath + 'app.js',
  appPath + 'app.constant.js',
  appPath + 'app.config.js',
  appPath + 'common/services/url.service.js',
  appPath + 'common/services/**/*.js',
  appPath + 'common/models/**/*.js',  	//model
  exculePath + 'common/**/ui-*.js', 		//排除所有ui文件不打包异步加载
  appPath + 'common/**/*.js',
  appPath + 'layout/**/*.js',				//布局
  appPath + '**/controller/*.js',           //每日数据controller
  appPath + 'app.router.js'			//路由
];

// css文件
var cssConfig = [
  libPath + 'bootstrap/css/bootstrap.min.css',
  libPath + 'font-awesome/font-awesome.min.css',
  libPath + 'simple-line-icons/simple-line-icons.min.css',
  libPath + 'loadingbar/loading-bar.min.css',
  //libPath + 'angular-ui-grid/build.css',
  libPath + 'angular-ag-grid/ag-grid.css',
  libPath + 'angular-ag-grid/theme-fresh.min.css',
  //libPath + 'angular-ui-layout/ui-layout.css',
  libPath + 'bootstrap-toastr/toastr.min.css',  
  cssPath + '*.css'  
];

var cssCopyConfig = ["!" + cssPath + '*.css', cssPath + '**/*.css'];

var templateConfig = [
  appPath + '**/modal/*.html',
  appPath + 'layout/*.html',
  appPath + '**/views/*.html'
];

// 除去appConfig的所有js文件
var scriptConfig = [
  appPath + '**/*.js',
  exculePath + '*.js',
  exculePath + 'layout/**/*.js',
  appPath + 'common/**/ui-*.js'
];

// html
var viewConfig = [
  "!" + appPath + '*/templates/*.html',
  appPath + '**/*.html'
];

gulp.copy=function(src, dest, base){
  base = base || '.';
  return gulp.src(src, {base: base})
    .pipe(gulp.dest(dest));
};

/*********
    合并，压缩css文件
********/
gulp.task('css', function() {
	gulp.src(cssConfig, srcConfig)
		.pipe(concat('app.css'))
		.pipe(cssmin())
		.pipe(rename({suffix: '.min'}))
		.pipe(gulp.dest(destPath + 'css'));
});
gulp.task('cssCopy', function() {
	gulp.src(cssCopyConfig, {base: './' + cssPath})
        .pipe(gulp.dest(destPath + 'css'));
});

/*********
    copy font
 ********/
gulp.task('font', function() {
  gulp.src([
    libPath + 'font-awesome/fonts/*',
    libPath + 'simple-line-icons/fonts/*',
    //libPath + 'angular-ui-grid/fonts/*'
  ])
    .pipe(rename({dirname: ''}))
    .pipe(gulp.dest(destPath + 'css/fonts'));
  
  gulp.src(libPath + 'bootstrap/fonts/bootstrap/*')
    .pipe(rename({dirname: ''}))
    .pipe(gulp.dest(destPath + 'fonts/bootstrap'));
});


/*********
 合并，压缩 基础的js  
 ********/
gulp.task('app', function() {
  gulp.src(appConfig, srcConfig)
    .pipe(concat('app.js', concatConfig))
    .pipe(rename({suffix: '.min'}))
    .pipe(gulp.dest(destPath + 'js'));
});

gulp.task('appMin', function() {
  gulp.src(appConfig, srcConfig)
    .pipe(concat('app.js', concatConfig))
    .pipe(uglify().on('error', function(e){
      console.log(e);
    }))
    .pipe(rename({suffix: '.min'}))
    .pipe(gulp.dest(destPath + 'js'))
});


// 打包初始化需要的库文件  不提供监控，直接压缩
gulp.task('lib', function() {
  gulp.src(libConfig, srcConfig)
    .pipe(concat('lib.js', concatConfig))
    .pipe(uglify().on('error', function(e){
      console.log(e);
    }))
    .pipe(rename({suffix: '.min'}))
    .pipe(gulp.dest(destPath + 'js'));
});

/*********
 转换固定模板为js  直接压缩
 ********/
gulp.task('template', function() {
  gulp.src(templateConfig)
    .pipe(html2js({
    	outputModuleName: 'templateModal',
    	useStrict: true,
    	rename: function (moduleName) {
    	  if (moduleName.indexOf('views') > -1) {
    	    return moduleName.replace('static/app/', 'views/');
	  } else if (moduleName.indexOf('layout') > -1) {
	    return moduleName.replace('static/app/', 'views/');
	  } else {
	    return moduleName.replace('static/app/', '').replace('modal', '');
	  }
	}
    }))
    .pipe(concat('template.js'))
    .pipe(uglify().on('error', function(e){
      console.log(e);
    }))
    .pipe(rename({suffix: '.min'}))
    .pipe(gulp.dest(destPath + 'js'));
});

/*********
复制文件
********/
// 复制转换 app目录下的文件到 dest下
gulp.task('copyView', function() {
  // html直接复制
  gulp.copy(viewConfig, destPath + 'views', appPath);
});

gulp.task('copyScript', function() {
  // js需要转换压缩
  gulp.src(scriptConfig, {base: appPath})
  //.pipe(ngAnnotate())
    .pipe(gulp.dest(destPath + 'js'));
});

gulp.task('copyScriptMin', function() {
  // js需要转换压缩
  gulp.src(scriptConfig, {base: appPath})
    .pipe(uglify().on('error', function(e){
      console.log(e);
    }))
    .pipe(gulp.dest(destPath + 'js'));
});

gulp.task('server', function() {
	browserSync.init({
        server: "./",
        //proxy: "http://www.oms.com:8080/",
        port: 8080
    });
    watchConfig(true);
})

// 默认任务
gulp.task('default', function() {
	watchConfig();
});

function watchConfig(isReload) {
	var watch = gulp.watch, 
		watchConfig = {debounceDelay: 2000},
		singleFileWatchConfig = extend(srcConfig, {base: appPath});
	gulp.run('css', 'cssCopy', 'template', 'app', 'font', 'copyScript');

	/**********
		监控 scriptconfig配置的js文件
	*********/
	watch(scriptConfig, watchConfig, function(e) {
		gulp.src(e.path, singleFileWatchConfig)
			//.pipe(ngAnnotate())
			.pipe(gulp.dest(destPath + 'js'))

		isReload && browserSync.reload();
	});
	// watch(viewConfig, watchConfig, function(e) {
	// 	gulp.src(e.path, singleFileWatchConfig)
	// 		.pipe(gulp.dest(destPath + 'views'))
	// 	isReload && browserSync.reload();
	// });
	watch(appConfig, watchConfig, function(e) {
		gulp.run('app');
		isReload && browserSync.reload();
	});

	watch(cssConfig, watchConfig, function(e) {
		gulp.run('css');
		isReload && browserSync.reload();
	});
	watch(cssCopyConfig, watchConfig, function(e) {
		gulp.run('cssCopy');
		isReload && browserSync.reload();
	});
	watch(templateConfig, watchConfig, function() {
		gulp.run('template');
		isReload && browserSync.reload();
	});
}

// 上线打包
gulp.task('p', function(){
  var watch = gulp.watch, watchConfig = {debounceDelay: 2000};
  gulp.run('lib', 'css', 'cssCopy', 'template', 'appMin', 'font', 'copyScriptMin');
});

function extend(targetObj, obj) {
	var resultObj = {};
	for (var key in targetObj) {
		resultObj[key] = targetObj[key];
	}
	for (var key in obj) {
		resultObj[key] = obj[key];
	}
	return resultObj;
}


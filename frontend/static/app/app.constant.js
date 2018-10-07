(function () {
    'use strict';

    var SCRIPT_PATH = 'js/';
    var VIEW_PATH = 'views/';
    var LIB_PATH = 'lib/';
    app.constant('SCRIPT_PATH', SCRIPT_PATH);
    app.constant('VIEW_PATH', VIEW_PATH);

    app.constant('APP_MEDIAQUERY', {
        'desktopXL': 1200,
        'desktop': 992,
        'tablet': 768,
        'mobile': 480
    });

    app.constant('JS_REQUIRES', {
        // Scripts
        scripts: {
            // Javascript 插件
            'perfect-scrollbar-plugin': [
                LIB_PATH + 'perfect-scrollbar/perfect-scrollbar.min.js',
                LIB_PATH + 'perfect-scrollbar/perfect-scrollbar.min.css'
            ],
            'ladda': [
                LIB_PATH + 'ladda/spin.js',
                LIB_PATH + 'ladda/ladda.min.js',
                LIB_PATH + 'ladda/ladda.min.css'
            ]
        },

        // angularJS 模块
        // name: module name
        // files: []
        modules: [{
                name: 'perfect_scrollbar',
                files: [LIB_PATH + 'angular-perfect-scrollbar/angular-perfect-scrollbar.js']
            }, {
                name: 'angular-ladda',
                files: [LIB_PATH + 'angular-ladda/angular-ladda.min.js']
            }, {
                name: 'ui.tree',
                files: [
                    LIB_PATH + 'jstree/css/default/style.min.css',
                    LIB_PATH + 'jstree/jstree.min.js',
                    SCRIPT_PATH + 'common/directives/ui-tree.directive.js'
                ]
            }, {
                name: 'ui.datepicker',
                files: [
                    LIB_PATH + 'bootstrap-datetimepicker/bootstrap-datetimepicker.min.css',
                    LIB_PATH + 'bootstrap-datetimepicker/bootstrap-datetimepicker.min.js',
                    SCRIPT_PATH + 'common/directives/ui-datepicker.directive.js'
                ]
            }, {
                name: 'daterangepicker',
                files: [
                    LIB_PATH + 'bootstrap-daterangepicker/daterangepicker-bs3.css',
                    LIB_PATH + 'bootstrap-daterangepicker/daterangepicker.js',
                    LIB_PATH + 'angular-daterangepicker/angular-daterangepicker.js'
                ]
            }, {
                name: 'angularMoment',
                files: [LIB_PATH + 'angular-moment/angular-moment.min.js']
            }, {
                name: 'ui.select2',
                files: [
                    LIB_PATH + 'select2/css/select2.min.css',
                    LIB_PATH + 'select2/css/select2-bootstrap.min.css',
                    LIB_PATH + 'select2/js/select2.full.min.js',
                    SCRIPT_PATH + 'common/directives/ui-select2.directive.js'
                ]
            }, {
                name: 'ui.maxlength',
                files: [
                    LIB_PATH + 'bootstrap-maxlength/bootstrap-maxlength.min.js',
                    SCRIPT_PATH + 'common/directives/ui-maxlength.directive.js'
                ]
            }, {
                name: 'ngAside',
                files: [LIB_PATH + 'angular-aside/angular-aside.min.js', LIB_PATH + 'angular-aside/angular-aside.min.css']
            }, {
                name: 'ui.ripple',
                files: [SCRIPT_PATH + 'common/directives/ui-ripple.directive.js']
            }, {
                name: 'ui.sortable',
                files: [LIB_PATH + 'jquery/jquery-ui.min.js', SCRIPT_PATH + 'common/directives/ui-sortable.directive.js']
            }, {
                name: 'jkuri.touchspin',
                files: [LIB_PATH + 'bootstrap-touchspin/jquery.bootstrap-touchspin.css', LIB_PATH + 'bootstrap-touchspin/jquery.bootstrap-touchspin.js', LIB_PATH + 'angular-touchspin/ngTouchSpin.js']
            }, {
                name: 'viewLayout',
                files: [SCRIPT_PATH + 'common/directives/view-layout.directive.js']
            },{
                name:'tableCsv',
                files: [LIB_PATH + 'tablecsv/tablecsv.js']
            }
        ]
    });
})();


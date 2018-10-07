'use strict';
// AppCtrl
app.controller('AppCtrl', ['$rootScope', '$scope', '$state', '$localStorage', '$window', '$document', '$timeout', 'cfpLoadingBar', 'VIEW_PATH', 'tools',

function ($rootScope, $scope, $state, $localStorage, $window, $document, $timeout, cfpLoadingBar, VIEW_PATH, tools) {
    $scope.viewPath = {
        header: VIEW_PATH + 'layout/header.html',
        sildebar: VIEW_PATH + 'layout/sidebar.html',
        content: VIEW_PATH + 'layout/main-content.html'
    };
    //$scope.access = tools.access.siderbar;
    var $win = $($window);

    $rootScope.$on('$stateChangeStart', function (event, toState, toParams, fromState, fromParams) {

        cfpLoadingBar.start();

    });
    $rootScope.$on('$stateChangeSuccess', function (event, toState, toParams, fromState, fromParams) {

        // 停止进度条
        event.targetScope.$watch("$viewContentLoaded", function () {

            cfpLoadingBar.complete();
        });

        // 切换页面是滚动到顶部
        $document.scrollTo(0, 0);
        $rootScope.currTitle = $state.current.title;
    });

    // state not found
    $rootScope.$on('$stateNotFound', function (event, unfoundState, fromState, fromParams) {
        console.log(unfoundState.to);
        console.log(unfoundState.toParams);
        console.log(unfoundState.options);
    });

    $rootScope.pageTitle = function () {
        return $rootScope.app.name + ' - ' + ($rootScope.currTitle || $rootScope.app.description);
    };

    $scope.event = {
        logout: function (argument) {
            tools.notification.confirm("是否真的要退出吗？").then(function() {
              
            });
        }
    };

    // 获取游览器的 viewport
    var viewport = function () {
        var e = window, a = 'inner';
        if (!('innerWidth' in window)) {
            a = 'client';
            e = document.documentElement || document.body;
        }
        return {
            width: e[a + 'Width'],
            height: e[a + 'Height']
        };
    };
    // 添加到scope上
    $scope.getViewPort = function () {
        return {
            'h': viewport().height,
            'w': viewport().width
        };
    };

    $win.on('resize', function () {
        $scope.$apply();
    });
}]);

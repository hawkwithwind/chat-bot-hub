(function() {
    "use strict";
    angular.module('ui.select2', [])
        .directive('uiSelect2', ['$timeout', function($timeout) {
        var options = {};
        return {
            require: 'ngModel',
            priority: 1,
            compile: function(tElem, tAttrs) {
                var watch,
                    repeatOption,
                    repeatAttr,
                    isSelect = tElem.is('select'),
                    isMultiple = angular.isDefined(tAttrs.multiple);

                // 如果有添加ng-repeat的option
                if (tElem.is('select')) {
                    repeatOption = tElem.find('option[ng-repeat]');

                    if (repeatOption.length) {
                        repeatAttr = repeatOption.attr('ng-repeat');
                        watch = jQuery.trim(repeatAttr.split('|')[0]).split(' ').pop();
                    }
                }

                return function(scope, elm, attrs, controller) {
                    // 获取配置文件
                    var opts = angular.extend({theme: 'bootstrap'}, options, scope.$eval(attrs.uiSelect2));

                    /*
                        转换为angular数据格式
                    */
                    var convertToAngularModel = function(select2_data) {
                        var model;
                        if (opts.simple_tags) {
                            model = [];
                            angular.forEach(select2_data, function(value, index) {
                                model.push(value.id);
                            });
                        } else {
                            model = select2_data;
                        }
                        return model;
                    };

                    /*
                        转换为select2数据格式
                    */
                    var convertToSelect2Model = function(angular_data) {
                        var model = [];
                        if (!angular_data) {
                            return model;
                        }

                        if (opts.simple_tags) {
                            model = [];
                            angular.forEach(angular_data, function(value, index) {
                                model.push({
                                    'id': value,
                                    'text': value
                                });
                            });
                        } else {
                            model = angular_data;
                        }
                        return model;
                    };

                    if (isSelect) {
                        delete opts.multiple;
                        delete opts.initSelection;
                    } else if (isMultiple) {
                        opts.multiple = true;
                    }
                    init();
                    if (controller) {
                        // 监控ngmodel
                        scope.$watch(tAttrs.ngModel, function(current, old) {
                            if (!current) {
                                return;
                            }
                            if (current === old) {
                                return;
                            }
                            controller.$render();
                        }, true);
                        // 页面渲染
                        controller.$render = function() {
                            if (isSelect) {
                                elm.val(controller.$viewValue)
                                $timeout(function() {
                                    elm.trigger('change');
                                })
                            } else {
                                if (opts.multiple) {
                                    controller.$isEmpty = function(value) {
                                        return !value || value.length === 0;
                                    };
                                    var viewValue = controller.$viewValue;
                                    if (angular.isString(viewValue)) {
                                        viewValue = viewValue.split(',');
                                    }
                                    elm.select2('data', convertToSelect2Model(viewValue));
                                    if (opts.sortable) {
                                        elm.select2("container").find("ul.select2-choices").sortable({
                                            containment: 'parent',
                                            start: function() {
                                                elm.select2("onSortStart");
                                            },
                                            update: function() {
                                                elm.select2("onSortEnd");
                                                elm.trigger('change');
                                            }
                                        });
                                    }
                                } else {
                                    if (angular.isObject(controller.$viewValue)) {
                                        elm.select2('data', controller.$viewValue);
                                    } else if (!controller.$viewValue) {
                                        elm.select2('data', null);
                                    } else {
                                        elm.select2('val', controller.$viewValue);
                                    }
                                }
                            }
                        };

                        // 如果有ng-repeat监控
                        if (watch) {
                            scope.$watch(watch, function(newVal, oldVal, scope) {
                                if (angular.equals(newVal, oldVal)) {
                                    return;
                                }
                                // Delayed so that the options have time to be rendered
                                $timeout(function() {
                                    elm.select2('val', controller.$viewValue);
                                    // Refresh angular to remove the superfluous option
                                    controller.$render();
                                    if (newVal && !oldVal && controller.$setPristine) {
                                        controller.$setPristine(true);
                                    }
                                });
                            });
                        }

                        // 在验证流程中更新class
                        controller.$parsers.push(function(value) {
                            var div = elm.prev();
                            div.toggleClass('ng-invalid', !controller.$valid)
                                .toggleClass('ng-valid', controller.$valid)
                                .toggleClass('ng-invalid-required', !controller.$valid)
                                .toggleClass('ng-valid-required', controller.$valid)
                                .toggleClass('ng-dirty', controller.$dirty)
                                .toggleClass('ng-pristine', controller.$pristine);
                            return value;
                        });

                        if (!isSelect) {
                            // Set the view and model value and update the angular template manually for the ajax/multiple select2.
                            elm.bind("change", function(e) {
                                e.stopImmediatePropagation();

                                if (scope.$$phase || scope.$root.$$phase) {
                                    return;
                                }
                                scope.$apply(function() {
                                    controller.$setViewValue(
                                    convertToAngularModel(elm.select2('data')));
                                });
                            });

                            if (opts.initSelection) {
                                var initSelection = opts.initSelection;
                                opts.initSelection = function(element, callback) {
                                    initSelection(element, function(value) {
                                        var isPristine = controller.$pristine;
                                        controller.$setViewValue(convertToAngularModel(value));
                                        callback(value);
                                        if (isPristine) {
                                            controller.$setPristine();
                                        }
                                        elm.prev().toggleClass('ng-pristine', controller.$pristine);
                                    });
                                };
                            }
                        }
                    }

                    elm.bind("$destroy", function() {
                        elm.select2("destroy");
                    });

                    attrs.$observe('disabled', function(value) {
                        elm.select2('enable', !value);
                    });

                    attrs.$observe('readonly', function(value) {
                        elm.select2('readonly', !! value);
                    });

                    if (attrs.ngMultiple) {
                        scope.$watch(attrs.ngMultiple, function(newVal) {
                            attrs.$set('multiple', !! newVal);
                            elm.select2(opts);
                        });
                    }

                    function init() {
                        elm.select2(opts);

                        elm.select2('data', controller.$modelValue);

                        //controller.$render();

                        // Not sure if I should just check for !isSelect OR if I should check for 'tags' key
                        if (!opts.initSelection && !isSelect) {
                            var isPristine = controller.$pristine;
                            controller.$pristine = false;
                            controller.$setViewValue(
                            convertToAngularModel(elm.select2('data')));
                            if (isPristine) {
                                controller.$setPristine();
                            }
                            elm.prev().toggleClass('ng-pristine', controller.$pristine);
                        }
                    }
                    
                };
            }
        };
    }]);
})();
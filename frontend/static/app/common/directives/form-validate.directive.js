(function () {
	'use strict';
	app
		.constant('validateConfig', {
			regex: {
				MOBILE_REG : /^(13[0-9]|15[012356789]|18[0-9]|14[57]|17[0-9])[0-9]{8}$/,
				ZIPCODE_REG : /[1-9]\d{5}/,
				IDCARD : /(^\d{15}$)|(^\d{17}(\d|X|x)$)/,
				EMAIL_REG : /^[\w\.-]+@[\w\.-]+\.\w{2,4}$/,
				NUMBER_REG :  /^-?\d+(.\d+)?$/,// /^\d+.?\d+$/,
				NAME: /^[\u4e00-\u9fa5]{2,15}$/,
				BANKCARD: /^\d{16}|\d{19}$/,
				CHINESE_REG: '/^[\u4e00-\u9fa5]+$/'
			},
			validateAttrs : [
				'required',
				'mobile',
				'email',
				'number',
				'ng-minlength',
				'ng-maxlength',
				'maxval',
				'minval',
				'ng-pattern',
				'interger',
				'money',
				'zipcode',
				'idcard',
				'requiredselect',
				'chinese'
			],
			defaultMsg  : {
				'required' : "该项不能为空",
				'mobile' : '手机号码格式不正确',
				'email' : '邮箱格式不正确',
				'number': '该项必须是数字',
				'ng-minlength': '该项不能小于{0}位数',
				'ng-maxlength': '该项不能大于{0}位数',
				'maxval': '该项数值不能大于{0}',
				'minval': '该项数值不能小于{0}',
				'ng-pattern': '该项不符合规则',
				'interger': '该项必须为整数',
				'money': '该项不符合规范',
				'requiredselect': "该项不能为空",
				'zipcode': "邮编格式不正确",
				'idcard': "身份证格式不正确",
				'chinese': '该项只能输入中文'
			},
			bindAttrs: ['maxval']
		})
		.directive('formValidate', formValidate)
		//email验证
		.directive("email", email)
		// 下拉框必填验证
		.directive("requiredselect", requiredselect)
		//最大值
		.directive("maxval", maxval)
		//最小值
		.directive("minval", minval)
		//手机号
		.directive("mobile", mobile)
		//手机号
		.directive("zipcode", zipcode)
		//手机号
		.directive("idcard", idcard)
		//数字验证
		.directive("number", number)
		//数字验证
		.directive("chinese", chinese)
		//货币验证
		.directive("money", money);
		formValidate.$inject = ['validateConfig'];
		function formValidate(validateConfig) {
			var result = {
				restrict: 'EA',
				compile: compile,
			};
			return result;

			function compile(elem, attrs) {
				elem.attr('novalidate', true);
				var formName = elem.attr('name'),
					$inputs = elem.find("input,textarea,select"), //获取验证的控件
					showErrorExpression = formName + '.showError',  //显示错误条件
					validateHtml = ''; //验证的html
				if (!formName) {
					throw new Error("form没指定name属性");
				}
				//遍历控件
				$inputs.each(function(v) {
					validateHtml = '';
					formatValidate($(this));
				});

				function formatValidate($input) {
					var inputType = $input.attr('type'),
						inputName = $input.attr("name"),
						nodeName = $input[0].nodeName,
						//错误消息
						errorMsg = "" ,
						// 创建元素出现错误条件
						elemValidateExpression = formName + '.' + inputName + '.$error';

					inputType = nodeName.toLowerCase();
					//没有name直接跳过
					if (!inputName) {return;}

					//遍历元素中配置的验证属性
					validateConfig.validateAttrs.forEach(function (validateAttr) {
						//获取验证属性
						var attrValue = $input.attr(validateAttr);
						//判断对象是否有验证属性
						if (typeof attrValue !== 'undefined' && attrValue !== false) {
							var errorMsg = getErrorMsg($input, attrValue, validateAttr);
							validateHtml += buildErrorElement(
								validateAttr,
								errorMsg
							);
						}
					});

					addDom(validateHtml, elemValidateExpression, $input);
				}

				function getErrorMsg ($input, attrValue, validateAttr) {
					var defaultErrorMsg = validateConfig.defaultMsg[validateAttr],
						attrValue = attrValue || '', //获取验证属性的值; 如 ng-minlength="2"
						errorMsg = '';
					// 是否有自定义错误消息
					errorMsg = $input.attr(validateAttr + '-error-msg') || defaultErrorMsg;
					if (validateConfig.bindAttrs.indexOf(validateAttr) != -1 ) {
						errorMsg = errorMsg.replace('{0}', '{{' + attrValue + '}}');
					} else {
						errorMsg = errorMsg.replace('{0}', attrValue);
					}
					return errorMsg;
				}
				function buildErrorElement (validateAttr, errorMsg) {
					// 去除ng-minlength和ng-maxlength前 的ng
					var validateAttr = validateAttr.indexOf('ng-') > -1 ? validateAttr.slice(3) : validateAttr;
					return '<div style="position:absolute;top:0" class="toggle" ng-message="' + validateAttr + '">' + errorMsg +'</div>';
				}
				function addHasErrorClassName(elemValidateExpression, $input) {
					// 替换$error 为 $invalid //没通过验证
					var errorExpression = showErrorExpression + " && " + elemValidateExpression.replace("$error", "$invalid");

					var $formGroup = $input.parent().attr('ng-class', "{'has-error': " + errorExpression + "}");
					//closest('.form-group')
				}
				function addDom ( validateHtml, elemValidateExpression, $input) {
					addHasErrorClassName(elemValidateExpression, $input);

					var html = '<div style="position:relative" class="text-small error" ng-messages="' + elemValidateExpression + '">' +
									'<div ng-if="' + showErrorExpression + '">' +
										validateHtml +
									'</div>' +
								'</div>';
					$input.parent().append(html);
				}
			}
		}

		email.$inject = ['validateConfig'];
		function email (validateConfig) {
			return {
				restrict: "A",
				require: 'ngModel',
				link: link
			};
			function link(scope, elem, attrs, ctrl) {
				ctrl.$parsers.push(function (v) {
					if (validateConfig.regex.EMAIL_REG.test(v)) {
						ctrl.$setValidity('email', true);
						return v;
					} else {
						ctrl.$setValidity('email', false);
						return undefined;
					}
				});
			}
		}

		function requiredselect() {
			return {
				restrict: "A",
				scope: {
					ngModel: '='
				},
				require: 'ngModel',
				link: link
			};

			function link (scope, elem, attrs, ctrl) {
				if (elem[0].nodeName == 'SELECT') {
					var val = scope.requiredselect || -1;
					scope.$watch('ngModel', function(v) {
						if (angular.isObject(v) && v.id === "") {
							ctrl.$setValidity('requiredselect', false);
							return v;
						} else {
							ctrl.$setValidity('requiredselect', true);
							return v;
						}
					});
				}
			}
		}

		function maxval() {
			return {
				restrict: "A",
				require: 'ngModel',
				scope: {
					maxval: '='
				},
				link: function (scope, elem, attrs, ctrl) {
					//判断是否是数字
					if (isNaN(scope.maxval)) {return false}
					ctrl.$parsers.push(function (v) {
						v = parseFloat(v);
						if (v <= scope.maxval) {
							ctrl.$setValidity('maxval', true);
							return v;
						} else {
							ctrl.$setValidity('maxval', isNaN(v));
							//ctrl.$setValidity('maxval', false);
							return undefined;
						}
					});
				}
			};
		}
		function minval() {
			return {
				restrict: "A",
				require: 'ngModel',
				link: function (scope, elem, attrs, ctrl) {
					//判断是否是数字
					var minVal = attrs['minval'];
					if (isNaN(minVal)) {return false;}
					ctrl.$parsers.push(function (v) {
						v = parseFloat(v);
						if (v >= minVal) {
							ctrl.$setValidity('minval', true);
							return v;
						} else {
							ctrl.$setValidity('minval', isNaN(v));
							return undefined;
						}
					});
				}
			};
		}
		mobile.$inject = ['validateConfig'];
		function mobile(validateConfig) {
			return {
				restrict: "A",
				require: 'ngModel',
				link: function (scope, elem, attrs, ctrl) {
					ctrl.$parsers.push(function (v) {
						if (validateConfig.regex.MOBILE_REG.test(v)) {
							ctrl.$setValidity('mobile', true);
							return v;
						} else {
							ctrl.$setValidity('mobile', false);
							return undefined;
						}
					});
				}
			};
		}
		zipcode.$inject = ['validateConfig'];
		function zipcode(validateConfig) {
			return {
				restrict: "A",
				require: 'ngModel',
				link: function (scope, elem, attrs, ctrl) {
					ctrl.$parsers.push(function (v) {
						if (validateConfig.regex.ZIPCODE_REG.test(v)) {
							ctrl.$setValidity('zipcode', true);
							return v;
						} else {
							ctrl.$setValidity('zipcode', false);
							return undefined;
						}
					});
				}
			};
		}

		idcard.$inject = ['validateConfig'];
		function idcard(validateConfig) {
			return {
				restrict: "A",
				require: 'ngModel',
				link: function (scope, elem, attrs, ctrl) {
					ctrl.$parsers.push(function (v) {
						if (validateConfig.regex.IDCARD.test(v)) {
							ctrl.$setValidity('idcard', true);
							return v;
						} else {
							ctrl.$setValidity('idcard', false);
							return undefined;
						}
					});
				}
			};
		}

		number.$inject = ['validateConfig'];
		function number (validateConfig) {
			return {
				restrict: "A",
				require: 'ngModel',
				link: function (scope, elem, attrs, ctrl) {
					ctrl.$parsers.push(function (v) {
						if (validateConfig.regex.NUMBER_REG.test(v)) {
							ctrl.$setValidity('number', true);
							return v;
						} else {
							ctrl.$setValidity('number', false);
							return undefined;
						}
					});
				}
			};
		}

		chinese.$inject = ['validateConfig'];
		function chinese (validateConfig) {
			return {
				restrict: "A",
				require: 'ngModel',
				link: function (scope, elem, attrs, ctrl) {
					ctrl.$parsers.push(function (v) {
						if (/^[\u4e00-\u9fa5]+$/.test(v)) {
							ctrl.$setValidity('chinese', true);
							return v;
						} else {
							ctrl.$setValidity('chinese', false);
							return undefined;
						}
					});
				}
			};
		}

		function money() {
			return {
				restrict: "A",
				require: 'ngModel',
				link: function (scope, elem, attrs, ctrl) {
					//判断是否是数字
					var val = attrs['money'];
					val = isNaN(val) ? 9 : val;
					val = val || 9;
					var reg = new RegExp('^\\d{1,' + val + '}(\\.\\d{0,4})?$');
					ctrl.$parsers.push(function (v) {
						if (reg.test(v)) {
							ctrl.$setValidity('money', true);
							return v;
						} else {
							ctrl.$setValidity('money', false);
							return undefined;
						}
					})
				}
			}
		}

})();

// 抛弃
angular.module('dh.formValidateDirective', [])
	.constant('regex', {
		MOBILE_REG : /^(13[0-9]|15[012356789]|18[0-9]|14[57]|17[0-9])[0-9]{8}$/,
		EMAIL_REG : /^[\w\.-]+@[\w\.-]+\.\w{2,4}$/,
		NUMBER_REG :  /^-?\d+(.\d+)?$/,// /^\d+.?\d+$/,
		NAME: /^[\u4e00-\u9fa5]{2,15}$/,
		BANKCARD: /^\d{16}|\d{19}$/
	})
	.constant('validate', {
		validateAttrs : [
			'required',
			'mobile',
			'email',
			'alipaycard',
			'username',
			'ng-minlength',
			'ng-maxlength',
			'number',
			'maxval',
			'minval',
			'ng-pattern',
			'requiredselect',
			'interger',
			'money'
		],
		defaultMsg  : {
			'required' : "该项不能为空",
			'number': '该项必须是数字',
			'email' : '邮箱格式不正确',
			'mobile' : '手机号码格式不正确',
			'username':'姓名应为等于或大于2位中文汉字',
			'alipaycard':'请输入手机号或者邮箱',
			'ng-minlength': '该项不能小于{0}位数',
			'ng-maxlength': '该项不能大于{0}位数',
			'maxval': '该项数值不能大于{0}',
			'minval': '该项数值不能小于{0}',
			'ng-pattern': '该项不符合规则',
			'money': '该项不符合规范',
			'interger': '该项必须为整数',
			'requiredselect': '该项不能为空',
			'custom': "该项不符合规则"
		}
	})
	//手机号
	.directive("mobile", ["regex", function (regex) {
		return {
			restrict: "A",
			require: 'ngModel',
			link: function (scope, elem, attrs, ctrl) {
				ctrl.$parsers.push(function (v) {
					if (regex.MOBILE_REG.test(v)) {
						ctrl.$setValidity('mobile', true);
						return v;
					} else {
						ctrl.$setValidity('mobile', false);
						return undefined;
					}
				})
			}
		}
	}])
	.directive("requiredselect", function () {
		return {
			restrict: "A",
			scope: {
				ngModel: '=',
				requiredselect: '='
			},
			require: 'ngModel',
			link: function (scope, elem, attrs, ctrl) {
				if (elem[0].nodeName == 'SELECT') {
					var val = scope.requiredselect || -1;
					scope.$watch('ngModel', function(v) {
						if (angular.isObject(v) && v.id == val) {
							ctrl.$setValidity('requiredselect', false);
							return v;
						} else {
							ctrl.$setValidity('requiredselect', true);
							return v;
						}
					})
				}
			}
		}
	})

	//数字验证
	.directive("number", ["regex", function (regex) {
		return {
			restrict: "A",
			require: 'ngModel',
			link: function (scope, elem, attrs, ctrl) {
				ctrl.$parsers.push(function (v) {
					if (regex.NUMBER_REG.test(v)) {
						ctrl.$setValidity('number', true);
						return v;
					} else {
						ctrl.$setValidity('number', false);
						return undefined;
					}
				})
			}
		}
	}])
	//整数
	.directive("interger", function () {
		return {
			restrict: "A",
			require: 'ngModel',
			link: function (scope, elem, attrs, ctrl) {
				ctrl.$parsers.push(function (v) {
					if(parseInt(v) == v){
						ctrl.$setValidity('interger', true);
						return parseInt(v);
					} else {
						ctrl.$setValidity('interger', false);
						return undefined;
					}
				})
			}
		}
	})
	//姓名
	.directive("username", ["regex", function (regex){
		return {
			restrict: "A",
			require: 'ngModel',
			link: function (scope, elem, attrs, ctrl) {
				ctrl.$parsers.push(function (v) {
					if (regex.NAME.test(v)||!v) {
						ctrl.$setValidity('username', true);
						return v;
					} else {
						ctrl.$setValidity('username', false);
						return undefined;
					}
				})
			}
		}
	}])
	//支付宝账号
	.directive("alipaycard", ["regex", function (regex){
		return {
			restrict: "A",
			require: 'ngModel',
			link: function (scope, elem, attrs, ctrl) {
				ctrl.$parsers.push(function (v) {
					if (regex.MOBILE_REG.test(v)||regex.EMAIL_REG.test(v)) {
						ctrl.$setValidity('alipaycard', true);
						return v;
					} else {
						ctrl.$setValidity('alipaycard', false);
						return undefined;
					}
				})
			}
		}
	}])
	//最大值
	.directive("maxval", function () {
		return {
			restrict: "A",
			require: 'ngModel',
			link: function (scope, elem, attrs, ctrl) {
				//判断是否是数字
				var maxVal = attrs['maxval'];
				if (isNaN(maxVal)) {return false}
				ctrl.$parsers.push(function (v) {
					v = parseFloat(v);
					if (v <= maxVal) {
						ctrl.$setValidity('maxval', true);
						return v;
					} else {
						ctrl.$setValidity('maxval', isNaN(v));
						//ctrl.$setValidity('maxval', false);
						return undefined;
					}
				})
			}
		}
	})
	//最小值
	.directive("minval", function () {
		return {
			restrict: "A",
			require: 'ngModel',
			link: function (scope, elem, attrs, ctrl) {
				//判断是否是数字
				var minVal = attrs['minval'];
				if (isNaN(minVal)) {return false}
				ctrl.$parsers.push(function (v) {
					v = parseFloat(v);
					if (v >= minVal) {
						ctrl.$setValidity('minval', true);
						return v;
					} else {
						ctrl.$setValidity('minval', isNaN(v));
						return undefined;
					}
				})
			}
		}
	})
	.directive("money",  function () {
		return {
			restrict: "A",
			require: 'ngModel',
			link: function (scope, elem, attrs, ctrl) {
				//判断是否是数字
				var val = attrs['money'];
				val = isNaN(val) ? 4 : val;
				val = val || 4;
				var reg = new RegExp('^-?\\d{1,' + val + '}(\\.\\d{0,2})?$');
				ctrl.$parsers.push(function (v) {
					if (reg.test(v)) {
						ctrl.$setValidity('money', true);
						return v;
					} else {
						ctrl.$setValidity('money', false);
						return undefined;
					}
				})
			}
		}
	})

	.directive('formValidate', ['validate', function(validate) {
		var formExpression = '';
		var buildExpression = function (formName, inputName, validateName, elem) {
			var formSubmit = formName + '.submit && ',
				expression = formName + '.' + inputName + '.$error.' + validateName;
			if (validateName != 'custom') {
				if (validateName.indexOf('ng-') > -1) {
					validateName = validateName.substr(3);
				}

				var customCondition = elem.attr('custom-condition');
				if (typeof customCondition !== 'undefined' && customCondition !== false) {
					expression += ' && ' + customCondition;
				}
			} else {
				expression = elem.attr('custom');
			}
			//添加form提交验证
			if (formExpression) {
				formExpression += '||' + expression;
			} else {
				//first
				formExpression = formSubmit + '(' + expression;
			}
			expression = formSubmit + expression;
			return expression;
		};
		var getErrorMsg = function($input, validateAttr) {
			var errorMsg = validate.defaultMsg[validateAttr];
			var validateVal = $input.attr(validateAttr); //获取验证属性的值;
			errorMsg = $input.attr(validateAttr + '-error-msg') || errorMsg;
			errorMsg = errorMsg.replace('{0}', validateVal);
			return errorMsg;
		};
		var buildElement = function (expression, errorMsg) {
			return '<span ng-if="' + expression + '" class="help-block animate-if">' + errorMsg + '</span>';
		};
		var addDom = function(elem, expression, errorMsg) {
			elem.after(buildElement(expression, errorMsg));
		};
		return {
			restrict: 'AE',
			compile: function(elem, attrs, trans) {
				elem.attr('novalidate', true);
				var formName = elem.attr('name'),
					$inputs = elem.find("input,textarea,select"); //获取验证的控件
				if (!formName) {
					throw new Error("form没指定name属性");
				}
				//遍历控件
				$inputs.each(function(v) {
					formExpression = '';
					var $input = $(this),
						inputType = $input.attr('type'),
						inputName = $input.attr("name"),
						nodeName = v.nodeName,
						validateAttr = '';
					if (nodeName == 'SELECT' || nodeName == 'TEXTAREA') {
						inputType = nodeName.toLowerCase();
					}
					//没有name直接跳过
					if (!inputName) {return;}
					//判断是否有自定义的验证  有自定义属性进行其他验证
					validateAttr = $input.attr('custom');
					if (typeof validateAttr !== 'undefined' && validateAttr !== false) {
						var expression = buildExpression(formName, inputName, 'custom', $input),
							errorMsg = getErrorMsg($input, 'custom');
						addDom($input, expression, errorMsg)
					} else {
						//遍历验证属性
						validate.validateAttrs.forEach(function (v) {
							var validateAttr = $input.attr(v);  //获取验证属性
							//判断对象是否有验证属性
							if (typeof validateAttr !== 'undefined' && validateAttr !== false) {
								var expression = buildExpression(formName, inputName, v, $input),
									errorMsg = getErrorMsg($input, v);
								addDom($input, expression, errorMsg)
							}
						});
					}
					if (formExpression) {
						formExpression += ')';
						$input.closest('.row').attr('ng-class', '{\'has-error\':' + formExpression + '}');
					}
				})

			}
		}
	}]);
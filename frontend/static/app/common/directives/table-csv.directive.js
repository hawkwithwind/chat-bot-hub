(function () {
    app.directive('tableCsv', tableCsv);
    function tableCsv() {
        return {
            link:function(scope,ele,attr){
                ele.click(function(){
                    var id = attr['sid'];
                    var text = attr['name'];
                    tableExport(id,text,'csv')
                })
            }
        }
    }
})();
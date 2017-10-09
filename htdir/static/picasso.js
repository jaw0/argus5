// Copyright (c) 2017
// Author: Jeff Weisberg <jaw @ tcp4me.com>
// Created: 2017-Oct-05 18:19 (EDT)
// Function: picasso paints graphs

var colors = ["#897fd9","#80ab51","#754192","#4bac73",
              "#c97dcb","#3c6e2c","#de77a4","#3fb5b1",
              "#8e396f","#81862f","#435193","#ac8a39",
              "#5890d7","#a95e24","#4d9fc1","#de9258",
              "#3e5676","#754618","#a2aad1","#625427",
              "#927eae","#93a36e","#c891b4","#3e8a78",
              "#724f66","#81aea3","#9d734b","#698599",
              "#b59e7d","#345d5c","#6e7d63","#425439"]

function configure_graphs(){

    $('.graph').each(function(){
        var el = $(this)
        var obj = el.attr("data-obj")
        makeGraph(el, obj)
    })
}

// fetch info - /api/graph

function makeGraph(el, obj){
    argus.log("graph " + obj)

    var g = { el: el, obj: obj, data:[] }

    $.ajax({
        dataType: "json",
        url: '/api/graph',
        data: {obj: obj},
        success: function(r){ fetch_graph_info(g, r)},
        error: ajax_fail
    });
}

var graphd
function fetch_graph_info(g, r){

    if( !r.graph ){
        argus.log("no graph info")
        return
    }

    // Title, YLabel, List[]{Obj, Label, Hwab, Tags[]
    g.info = r.graph
    graphd = g

    // fetch graph data
    var i
    var L = g.info.List
    for(i=0; i<L.length; i++){
        argus.log("graph + " + L[i].Obj)
        graph_fetch_data(g, i, 'samples', '')
    }
}
function graph_fetch_data(g, i, which, tag){

    $.ajax({
        dataType: "json",
        url: '/api/graphd',
        data: {obj: g.info.List[i].Obj, which: which, tag: tag, width: g.el.width() },
        success: function(r){ build_graph_ok(g, i, which, tag, r)},
        error: ajax_fail
    });
}


function build_graph_ok(g, i, which, tag, r){

    argus.log("graph data ok " )

    // XXX - {obj, which, tag}
    g.data[i] = r.data

    var plot = { data: graphValue(g.data[i]),
                 color: '#adc',
                 label: 'label'
               }
    var opts = {
        xaxis: {mode: "time"}
    }


}


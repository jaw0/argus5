// Copyright (c) 2017
// Author: Jeff Weisberg <jaw @ tcp4me.com>
// Created: 2017-Oct-05 18:19 (EDT)
// Function: picasso paints graphs


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

// https://github.com/flot/flot/blob/master/API.md
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
    
    $.plot(g.el, [plot], opts );

}


//****************************************************************

function graphValue(d){

    var i
    var a = []
    for(i=0; i<d.length; i++){
        // RSN - missing value? push null
        a.push( [d[i].When * 1000, d[i].Value] )
    }

    return a
}

function graphHwab(d){

    var i
    var a = []
    for(i=0; i<d.length; i++){
        a.push( [d[i].When * 1000, d[i].Exp + d[i].Delt, d[i].Exp - d[i].Delt] )
    }

    return a
}


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
        var el    = $(this)
        var obj   = el.attr("data-obj")
        var which = el.attr("data-which") || 'samples'
        var darp  = el.attr("data-darp")
        var ctls  = el.attr("data-ctls")

        var wid   = el.width()
        new Graph(el.get(0), obj, which, darp, ctls, wid)
    })
}

var graphd	// for debugging

function Graph(el, obj, which, darp, ctls, width){

    argus.log("new graph " + obj)
    this.el    = el
    this.obj   = obj
    this.which = which	// to initially load
    this.darp  = darp
    this.ctls  = ctls   // show controls
    this.width = width
    this.selected = {}
    this.pending  = {}
    this.datasets = {}
    this.coloridx = 0

    this.fetchGraphInfo()
    graphd = this
    return this
}

(function(){
    var p = Graph.prototype

    p.fetchGraphInfo = function(){

        var g = this
        $.ajax({
            dataType: "json",
            url: '/api/graph',
            data: {obj: this.obj},
            success: function(r){ g.gotGraphInfo(r) },
            error: function(a,b,c){   g.ajaxFail(a,b,c) }
        });
    }

    p.ajaxFail = function(r,err,msg){
        argus.log("error: " + msg)
        this.el.innerHTML = 'ERROR: cannot load graph info: ' + msg
    }

    p.gotGraphInfo = function(r){

        argus.log("graph info")
        if( !r.graph ){
            argus.log("no graph info")
            return
        }

        // Title, YLabel, MyId, List[]{Obj, Label, Hwab, Tags[]
        this.info = r.graph

        // create chart
        this.cs   = new ChartStrip(this.el, {
            title:	 this.info.Title,
            ylabel:	 this.info.YLabel,
            draw_border: 0
        })

        // gather darp tag list
        // what type of graph?
        // assign colors
        this.learnObjs()
        // create selector - range + darp + objs
        this.createControls()
        // select + fetch
        this.selectAll(this.which, this.darp || this.info.MyId)
    }

    p.learnObjs = function(){
        var darp = {}
        var i, j
        var L = this.info.List

        this.objs = {}

        for(i=0; i<L.length; i++){
            for(j=0; j<L[i].Tags.length; j++){
                darp[ L[i].Tags[j] ] = 1
                L[i].color = colors[i]
                this.objs[ L[i].Obj ] = L[i]
            }
        }
        this.darptags = Object.keys(darp)
    }

    p.createControls = function(){
        var div = document.createElement('div')
        div.className = 'graphcontrols'
        // insertAfter
        this.el.parentNode.insertBefore(div, this.el.nextSibling)


        if( this.ctls ){
            // range
            // darp
            div.innerHTML = '<div class=graphrange><input type=radio name=range value=samples>Day<br>' +
                '<input type=radio name=range value=hours>Week<br>' +
                '<input type=radio name=range value=days>Year<br>' +
                '</div>' +
                '<div class=graphdarp>ccsphl<br>qtssjc<br></div>' +
                '<br style="clear:both;">'
        }
        // labels

    }

    p.Id = function(which, darp, obj){
        return which + " " + darp + " " + obj
    }

    p.selectAll = function(which, darp){
        var i
        var L = this.info.List

        for(i=0; i<L.length; i++){
            argus.log("graph + " + L[i].Obj)
            // RSN - check obj has this darp tag
            // RSN - maybe fetch
            this.fetchData(which, darp, L[i].Obj)
            this.selected[ this.Id(which, darp, L[i].Obj) ] = 1
        }
    }

    p.fetchData = function(which, darp, obj){
        var g = this

        this.pending[ this.Id(which, darp, obj) ] = 1

        // fetch the graph data
        $.ajax({
            dataType: "json",
            url: '/api/graphd',
            data: {obj: obj, which: which, tag: darp, width: this.width },
            success: function(r){ g.gotData(which, darp, obj, r)},
            error:   function(r){ g.gotFail(which, darp, obj, r) }
        });
    }

    p.gotData = function(which, darp, obj, r){
        var id = this.Id(which, darp, obj)
        delete this.pending[id]
        this.datasets[id] = r.data

        // add to chart
        this.cs.Add( r.data, {
            id:		id,
            color:	this.objs[obj].color,
            smooth:	1,
            type:	'line',
            shadow:	1
            // ...
        })

        if( Object.keys(this.pending).length == 0 ){
            this.build()
        }
    }
    p.gotFail = function(which, darp, obj, r){
        var id = this.Id(which, darp, obj)
        delete this.pending[id]

        if( Object.keys(this.pending).length == 0 ){
            this.build()
        }
    }

    p.build = function(){
        this.cs.Render()
    }

})()




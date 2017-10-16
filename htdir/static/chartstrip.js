// Copyright (c) 2017
// Author: Jeff Weisberg <jaw @ tcp4me.com>
// Created: 2017-Oct-08 16:32 (EDT)
// Function: draw strip charts
//
// see Chart::Strip


function ChartStrip(el, opts){

    this.el = el
    this.defaultOpts(opts)
    this.init()

    return this
}

(function(){
    var p = ChartStrip.prototype

    p.defaults = {
        title_font: 	   '16px Arial, Sans-serif',
        title_color: 	   '#432',
        ylabel_font: 	   '14px Arial, Sans-serif',
        ylabel_color: 	   '#432',
        branding: 	   '',	// Aldeborontiphoscophornio!
        branding_font:     '10px Monospace',
        branding_color:    '#8AE',
        axii_color: 	   '#432',
        border_color:	   '#bbb',
        limit_factor: 	   4,
        smooth_factor: 	   1, // best results: 0.75 - 1
        plot_line_thick:   3,
        n_y_tics:          4, // aprox.
        binary:		   0,
        draw_border:       1,
        draw_grid:         1,
        draw_tics:         1,
        draw_tic_labels:   1,
        xtic_color:        '#432',
        ytic_color:	   '#432',
        grid_color:	   '#432',
        xtic_label_color:  '#432',
        mark_label_color:  '#d44',
        xtic_label_font:   'bold 12px Monospace',
        ytic_label_color:  '#432',
        ytic_label_font:   'bold 12px Monospace',

        comma: ','
    }

    p.defaultOpts = function(uopts){
        var opts = {}
        var i
        // merge user supplied opts with defaults, above
        var k = Object.keys(this.defaults)
        for(i=0; i<k.length; i++){
            opts[ k[i] ] = this.defaults[ k[i] ]
        }
        k = Object.keys(uopts)
        for(i=0; i<k.length; i++){
            opts[ k[i] ] = uopts[ k[i] ]
        }

        this.opts = opts
    }

    p.init = function(){
        var g = this
        var c = document.createElement('canvas')
        c.width = this.el.scrollWidth
        c.height = this.el.scrollHeight
        this.el.appendChild(c)
        this.C = c.getContext("2d")
        this.width = c.scrollWidth
        this.height = c.scrollHeight
        this.datasets = []
        this.margin_top = this.margin_bottom = this.margin_left = this.margin_right = 0

        this.drawLoading()
    }


    p.Add = function(data, opts){
        if( !('data_func'  in opts) ) opts.data_func  = this.AsIs
        if( !('color_func' in opts) ) opts.color_func = this.Color
        if( !('type' in opts) ) opts.type = 'line'

        var set = {data: data, opts: opts}

        this.analyze( set )
        this.datasets.push( set )
    }

    p.Replace = function(id, data){

        for(i=0; i<this.datasets.length; i++){
            if( this.datasets[i].opts.id != id ) continue

            this.datasets[i].data = data
            this.analyze( this.datasets[i] )
        }
    }

    p.Hide = function(id){
        var i

        for(i=0; i<this.datasets.length; i++){
            if( this.datasets[i].opts.id == id ) this.datasets[i].hide = 1
        }
    }
    p.HideAll = function(){
        var i

        for(i=0; i<this.datasets.length; i++){
            this.datasets[i].hide = 1
        }
    }
    p.Show = function(id){
        var i

        for(i=0; i<this.datasets.length; i++){
            if( this.datasets[i].opts.id == id ) this.datasets[i].hide = 0
        }
    }

    // I hear beyond the range of sound,
    // I see beyond the range of sight,
    // New earths and skies and seas around,
    // And in my day the sun doth pale his light.
    //   -- Thoreau, Inspiration

    p.analyze = function(dataset){
        var xmin, xmax, ymin, ymax
        var i, p, left, right, dxl, dxr, dyl, dyr, dl, dr
        var dydx = []

        var data = dataset.data

        if( !data ) return

        for(i=0;  i<data.length; i++){
            p = dataset.opts.data_func( data[i] )
            if( !p ) continue

            if( typeof(xmin) == 'undefined' ) xmin = p.Time
            xmax = p.Time

            //console.log("analyze: " + i + ": " + p.Time + " " + p.Value )

            if( dataset.opts.type == 'range' ){
                if( i == 0 ){
                    ymin = p.Min
                    ymax = p.Max
                }
                if( p.Min < ymin ) ymin = p.Min
                if( p.Max > ymax ) ymax = p.Max
            }
            if( dataset.opts.type == 'line' ){
                if( i == 0 ) ymin = ymax = p.Value
                if( p.Value < ymin ) ymin = p.Value
                if( p.Value > ymax ) ymax = p.Value

                // calc derivative
                left  = (i==0) ? p : dataset.opts.data_func( data[i-1] )
                right = (i==data.length-1) ? p : dataset.opts.data_func( data[i+1] )

                dxl = p.Time - left.Time
                dxr = right.Time - p.Time
                dyl = p.Value - left.Value
                dyr = right.Value - p.Value

                if( dxl && dxr ){
                    dl = dyl / dxl
                    dr = dyr / dxr

                    // mathematicaly, (dl+dr)/2 is the best estimate of the derivative,
                    // and gives the smoothest curve
                    // but, this way looks nicer...

                    if( dyl * dyr < 0 ){
                        // local extrema
                        // we do not want to over/under shoot
                        dydx[i] = 0
                    }else if( Math.abs(dl) < Math.abs(dr) ){
                        dydx[i] = 0.75 * dl + 0.25 * dr
                    }else{
                        dydx[i] = 0.75 * dr + 0.25 * dl
                    }

                }else if( dxr ){
                    dydx[i] = dyr / dxr
                }else if( dxl ){
                    dydx[i] = dyl / dxl
                }
            }
        }
        dataset.yd_min = ymin
        dataset.yd_max = ymax
        dataset.xd_min = xmin
        dataset.xd_max = xmax
        dataset.dydx   = dydx
    }

    p.adjust = function(sets){
        var xmin, xmax, ymin, ymax
        var i,s

        // determine min/max from selected datasets
        for(i=0; i<sets.length; i++){
            s = sets[i]

            if( i == 0 ){
                xmin = s.xd_min
                xmax = s.xd_max
                ymin = s.yd_min
                ymax = s.yd_max
            }

            if( s.xd_min < xmin ) xmin = s.xd_min
            if( s.xd_max > xmax ) xmax = s.xd_max
            if( s.yd_min < ymin ) ymin = s.yd_min
            if( s.yd_max > ymax ) ymax = s.yd_max
        }

        this.xd_min = xmin
        this.xd_max = xmax
        this.yd_min = ymin
        this.yd_max = ymax
    }

    p.setScale = function(){

        // I have touched the highest point of all my greatness;
        //   -- Shakespeare, King Henry VIII
        this.xmax = this.width  - this.margin_right  - this.margin_left
        this.ymax = this.height - this.margin_bottom - this.margin_top

        this.xd_scale = (this.xd_max == this.xd_min) ? 1 : this.xmax / (this.xd_max - this.xd_min)
        this.yd_scale = (this.yd_max == this.yd_min) ? 1 : this.ymax / (this.yd_max - this.yd_min)
    }

    p.Render = function(){
        // RSN - figure out which datasets
        var sets = []
        var i

        for(i=0; i<this.datasets.length; i++){
            if( this.datasets[i].hide ) continue
            sets.push( this.datasets[i] )
        }

        this.margin_top = this.margin_bottom = this.margin_left = this.margin_right = 0
        this.C.clearRect(0,0, this.width, this.height)
        this.drawBorder()
        this.drawLabels()
        this.adjust(sets)
        this.determineYtics()
        this.determineXtics()
        this.setScale()

        // plot graphs
        var i
        for(i=0; i<sets.length; i++){
            if( sets[i].opts.type == 'range' ){
                this.plot_range(sets[i])
            }
        }
        // boxes, points?
        for(i=0; i<sets.length; i++){
            if( sets[i].opts.type == 'line' ){
                this.plot_line(sets[i])
            }
        }

        this.drawAxii()
        this.drawGrid()
    }

    p.px = function(x){
        return x + this.margin_left
    }
    p.py = function(y){
        // make 0 the bottom
        return this.height - y - this.margin_bottom
    }
    p.dx = function(x){
        return this.px( (x - this.xd_min) * this.xd_scale )
    }
    p.dy = function(y){
        if( y < this.yd_min ) y = this.yd_min
        if( y > this.yd_max ) y = this.yd_max
        return this.py( (y - this.yd_min) * this.yd_scale )
    }

    p.drawBorder = function(){
        if( this.opts.draw_border ){
            this.C.strokeStyle = this.opts.border_color
            this.C.strokeRect(0,0, this.width, this.height)
        }
    }

    p.drawLoading = function(){
        var sz = 20
        var X = Math.ceil(this.width / sz)
        var Y = Math.ceil(this.height / sz)
        var x, y

        this.C.save()
        this.C.fillStyle = '#f8f8f8'

        for(y=0; y<Y; y++){
            for(x=0; x<X; x++){
                if( (x^y)&1 ) this.C.fillRect( x*sz, y*sz, sz, sz )
            }
        }

        this.C.strokeStyle = '#eee'
        this.C.lineWidth = 20

        this.C.beginPath()
        this.C.moveTo(this.width * .05, this.height * .1)
        this.C.lineTo(this.width * .05, this.height * .9)
        this.C.lineTo(this.width * .95, this.height * .9)
        this.C.stroke()

        this.C.moveTo(this.width * .05, this.height * .8)
        this.C.lineTo(this.width * .30, this.height * .5)
        this.C.lineTo(this.width * .50, this.height * .6)
        this.C.lineTo(this.width * .95, this.height * .1)
        this.C.stroke()

        this.C.fillStyle = '#eee'
        this.C.font = 'bold 50px Monospace'
        w = Math.round(this.C.measureText(this.opts.title).width)
        h = Math.round(this.C.measureText('m').width * 1.25)
        this.C.fillText(this.opts.title, Math.round((this.width - w)/2), h)

        this.C.restore()
    }

    p.drawLabels = function(){
        var w, h

        // title
        if( this.opts.title ){
            this.C.fillStyle = this.opts.title_color
            this.C.font = this.opts.title_font
            w = Math.round(this.C.measureText(this.opts.title).width)
            h = Math.round(this.C.measureText('m').width * 1.25)
            this.C.fillText(this.opts.title, Math.round((this.width - w)/2), h)
            this.margin_top = h + 5
        }
        // ylabel
        if( this.opts.ylabel ){
            this.C.fillStyle = this.opts.ylabel_color
            this.C.font = this.opts.ylabel_font
            w = Math.round(this.C.measureText(this.opts.ylabel).width)
            h = Math.round(this.C.measureText('m').width * 1.25)

            this.C.save()
            this.C.translate(0,this.height)
            this.C.rotate(-Math.PI/2)
            this.C.fillText(this.opts.ylabel, Math.round((this.height - w)/2), h)
            this.C.restore()

            this.margin_left = h + 5
        }
        // url
        if( this.opts.branding ){
            this.C.fillStyle = this.opts.branding_color
            this.C.font = this.opts.branding_font
            w = Math.round(this.C.measureText(this.opts.branding).width)
            h = Math.round(this.C.measureText('m').width * 1.25)

            this.C.save()
            this.C.rotate(Math.PI/2)
            this.C.translate(0,-this.width)
            this.C.fillText(this.opts.branding, 10, h)
            this.C.restore()

            this.margin_right = h + 5
        }
    }

    p.AsIs = function(p){
        return p
    }
    p.Color = function(p, c){
        return p['Color'] || c
    }

    p.plot_line = function(set){
        if( !set.data ) return
        var C = this.C
        var data = set.data
        var dydx = set.dydx
        var color = set.opts.color
        var dfunc = set.opts.data_func
        var cfunc = set.opts.color_func
        var limit = this.opts.limit_factor * (this.xd_max - this.xd_min) / data.length
        var smooth = set.opts.smooth ? this.opts.smooth_factor : 0
        var prevcolor
        var i, c, p, pp, gap
        var dxt, cx0, cx1, xy0, cy1

        C.save()
        C.lineWidth = this.opts.plot_line_thick
        C.lineJoin  = 'round'
        if( set.opts.shadow ){
            C.shadowColor = '#ccc'
            C.shadowOffsetX = 3
            C.shadowOffsetY = 3
            C.shadowBlur = 5
        }

        C.beginPath()

        for(i=0; i<data.length; i++){
            p = dfunc( data[i] )
            if( !p ) continue
            c = cfunc( data[i], color )

            gap = ( pp && limit && (p.Time - pp.Time) > limit )
            if( gap )  pp = undefined

            if( pp ){
                if( smooth ){
                    // pick bezier control points
                    //   smooth = (.5 - 1) gives nice curves
                    //   smooth > 1 gives straighter segments
                    //   smooth <= .5 takes the graph on a drug trip
                    dxt = (p.Time - pp.Time) / (smooth * 3)
                    cx0 = pp.Time + dxt
                    cx1 = p.Time  - dxt
                    cy0 = pp.Value + dydx[i-1] * dxt
                    cy1 = p.Value  - dydx[i] * dxt

                    C.bezierCurveTo(this.dx(cx0), this.dy(cy0), this.dx(cx1), this.dy(cy1), this.dx(p.Time), this.dy(p.Value))
                }else{
                    C.lineTo( this.dx(p.Time), this.dy(p.Value) )
                }
            }
            if( (c != prevcolor) || !pp ){
                C.stroke()
                C.beginPath()
                C.strokeStyle = c
                C.moveTo( this.dx(p.Time), this.dy(p.Value) )
            }

            prevcolor = c
            pp = p
        }
        C.stroke()
        C.restore()
    }


    p.plot_range = function(set){
        var C = this.C
        var data = set.data
        var color = set.opts.color
        var dfunc = set.opts.data_func
        var cfunc = set.opts.color_func
        var limit = this.opts.limit_factor * (this.xd_max - this.xd_min) / data.length
        var prevcolor
        var i, c, p, pp, gap

        C.save()
        C.lineWidth = 0

        for(i=0; i<data.length; i++){
            p = dfunc( data[i] )
            if( !p ) continue
            c = cfunc( data[i], color )

            gap = ( pp && limit && (p.Time - pp.Time) > limit )
            if( gap ){
                pp = undefined
            }
            C.fillStyle = c

            if( pp ){
                C.beginPath()
                C.moveTo( this.dx(pp.Time), this.dy(pp.Min) )
                C.lineTo( this.dx(pp.Time), this.dy(pp.Max) )
                C.lineTo( this.dx(p.Time),  this.dy(p.Max) )
                C.lineTo( this.dx(p.Time),  this.dy(p.Min) )
                C.closePath()
                C.fill()
            }

            pp = p
        }
        C.restore()
    }

    p.pretty = function(y, st){
        var ay, sc, b, prec

        sc = ''
        ay = Math.abs(y)
        b = this.opts.binary ? 1024 : 1000

        if( ay < 1 ){
	    if( ay < 1/Math.pow(b,3) ){
	        return "0";
	    }else if( ay < 1/Math.pow(b,2) ){
	        y *= Math.pow(b, 3); st *= Math.pow(b, 3);
	        sc = 'n';
	    }else if( ay < 1/b ){
	        y *= Math.pow(b,2); st *= Math.pow(b,2);
	        sc = 'u';
	    }else if( ay < 100/b ){
	        y *= b; st *= b;
	        sc = 'm';
	    }
        }else{
	    if( ay >= Math.pow(b,4) ){
	        y /= Math.pow(b,4);  st /= Math.pow(b,4);
	        sc = 'T';
	    }else if( ay >= Math.pow(b,3) ){
	        y /= Math.pow(b,3);  st /= Math.pow(b,3);
	        sc = 'G';
	    }else if( ay >= Math.pow(b,2) ){
	        y /= Math.pow(b,2); st /= Math.pow(b,2);
	        sc = 'M';
	    }else if( ay >= b ){
	        y /= b;   st /= b;
	        sc = 'k';
	    }
        }
        if( sc && this.opts.binary ){
            sc += 'i' // as per IEC 60027-2
        }
        if( st > 1 ){
	    prec = 0;
        }else{
	    prec = Math.abs(Math.floor(Math.log10(st)));
        }

        // my castle for sprintf
        if( !y ) y = 0
        var i = '' + Math.floor(y)
        var l = i.length
        return "" + y.toPrecision(prec+l) + sc
    }

    p.determineYtics = function(){
        var min = this.yd_min
        var max = this.yd_max
        var maxw = 0
        var lb, w, tp, is, st, low, i, y
        var tics = []

        this.C.font = this.opts.ytic_label_font
        var ht = this.C.measureText("m").width * 1.25

        if( min == max ){
            // not a very interesting graph...
            lb = this.pretty(min, 1)
            w = this.C.measureText(lb).width
            tics.push( { y: min, label: lb, width: w, height: ht } )
            maxw = w
        }else{
            tp = (max - min) / this.opts.n_y_tics // approx spacing of tics
            if( this.opts.binary ){
                is = Math.pow(2, Math.floor( Math.log(tp)/Math.log(2) ))
            }else{
                is = Math.pow(10, Math.floor( Math.log10(tp) ))
            }

            st = Math.floor( tp/is ) * is	// between 4-8
            if( st == 0 ) st = is
            low = Math.floor( min / st ) * st

            for(i=0; i<2*this.opts.n_y_tics+2; i++){
                y = low + i * st
                if( y >= max ) break
                if( y < min ) continue
                lb = this.pretty(y, st)
                w = this.C.measureText(lb).width
                if( w > maxw ) maxw = w
                tics.push( {y: y, label: lb, width: w, height: ht} )
            }
        }

        if( this.opts.draw_tic_labels ){
            // move margin
            this.margin_left += maxw + 10
        }

        this.ytics = tics
    }

    p.xtic_range_data = function(range){
        var range_hrs = range / 3600
        var range_days = range_hrs / 24

        // return: step, labeltype, marktype, lti, tmod

        if( range < 720 ){
	    return [60, 'HM', 'HR', 'min', 1]		// tics: 1 min
        }else if( range < 1800 ){
	    return [300, 'HM', 'HR', 'min', 5]		// tics: 5 min
        }else if( range_hrs < 2 ){
	    return [600, 'HM', 'HR', 'min', 10]		// tics: 10 min
        }else if( range_hrs < 6 ){
	    return [1800, 'HR', 'MN', 'min', 30]	// tics: 30 min
        }else if( range_hrs < 13 ){
	    return [3600, 'HR', 'MN', 'hour', 1]	// tics: 1 hr
        }else if( range_hrs < 25 ){
	    return [3600, 'HR', 'MN', 'hour', 2]	// tics: 2 hrs
        }else if( range_hrs < 50 ){
	    return [3600, 'HR', 'MN', 'hour', 4]	// tics: 4 hrs
        }else if( range_hrs < 75 ){
	    return [3600, 'HR', 'MN', 'hour', 6]	// tics: 6 hrs
        }else if( range_days < 15 ){
	    return [3600*24, 'DW', 'SU', 'day', 1]	// tics 1 day
        }else if( range_days < 22 ){
	    return [3600*24, 'DM', 'M1', 'day', 2]	// tics: 2 days
        }else if( range_days < 80 ){
	    return [3600*24, 'DM', 'M1', 'day', 7]	// tics: 7 days
        }else if( range_days < 168 ){
	    return [3600*24, 'DM', 'Y1', 'day', 14]	// tics: 14 days
        }else if( range_days < 370 ){
	    return [3600*24*31, 'DM', 'Y1', 'mon', 1]	// tics: 1 month
        }else if( range_days < 500 ){
	    return [3600*24*31, 'DM', 'Y1', 'mon', 2]	// tics: 2 month
        }else if( range_days < 1000 ){
	    return [3600*24*31, 'DM', 'Y1', 'mon', 3]	// tics: 3 month
        }else if( range_days < 2000 ){
	    return [3600*24*31, 'DM', 'NO', 'mon', 6]	// tics: 6 month
        }else{
	    return [3600*24*366, 'YR', 'NO', 'mon', 12]	// tics: 1 yr
        }
    }

    p.xtic_align_initial = function(step){
        var t = (step < 3600) ? Math.floor(this.xd_min / step) * step : (Math.floor(this.xd_min / 3600) * 3600)
        var lt, dt

        if( step >= 3600*24*365 ){
            while(1){
                // search for 1jan
                lt = new Date(t * 1000)
                if( (lt.getMonth() == 0) && (lt.getDate() == 1) && (lt.getHours() == 0) ) break
                // jump fwd: 1M, 1D, or 1H
                dt = (lt.getMonth() == 11) ? 24*30 : (lt.getDate() < 30) ? 24 : 1
                t += dt * 3600
            }

        }else if( step >= 3600*24*31 ){
	    while(1){
	        // find 1st of mon
                lt = new Date(t * 1000)
                if( (lt.getDate() == 1) && (lt.getHours() == 0) ) break
                dt = (lt.getDate() < 28) ? 24 : 1
                t += dt * 3600
            }

        }else if( step >= 3600*24 ){
            argus.log("t: " + t)
	    while(1){
                // search for midnight
                lt = new Date(t * 1000)
                if( lt.getHours() == 0 ) break
	        t += 3600;
	    }
        }

        return t
    }

    p.localtime = function(t){
        var lt = new Date(t * 1000)
        return {
            sec:  lt.getSeconds(),
            min:  lt.getMinutes(),
            hour: lt.getHours(),
            day:  lt.getDate(),
            mon:  lt.getMonth(),
            year: lt.getFullYear(),
            dow:  lt.getDay()
        }
    }

    p.format_time = function(rlt, labtyp, redmark){
        var MONTH = ["Jan", "Feb", "Mar", "Apr", "May", "Jun", "Jul", "Aug", "Sep", "Oct", "Nov", "Dec"]
        var DAY   = ["Sun", "Mon", "Tue", "Wed", "Thu", "Fri", "Sat"]
        var min = rlt.min
        if( min < 10 ) min = '0' + min

	if( labtyp == 'HM' ){
            return '' + rlt.hour + ':' + min 			// time
	}
	if( labtyp == 'HR' ){
	    if( redmark ){
                return '' + rlt.day + '/' + MONTH[rlt.mon]	// date DD/Mon
	    }else{
                return '' + rlt.hour + ':' + min 		// time
	    }
	}
	if( labtyp == 'DW' ){
	    if( redmark ){
                return '' + rlt.day + '/' + MONTH[rlt.mon]	// date DD/Mon
	    }else{
                return DAY[rlt.dow]				// day of week
	    }
	}
	if( labtyp == 'DM' ){
	    if( !rlt.day && !rlt.mon ){
                return rlt.year					// year
	    }else{
                return '' + rlt.day + '/' + MONTH[rlt.mon]	// date DD/Mon
	    }
	}
	if( labtyp == 'YR' ){
            return rlt.year					// year
	}

    }

    // this is good for (roughly) 10 mins - 10 yrs
    p.determineXtics = function(){
        if( this.xd_max == this.xd_min ) return
        var range      = this.xd_max - this.xd_min
        var range_hrs  = range / 3600
        var range_days = range_hrs / 24

        var xrd = this.xtic_range_data(range)
        var step=xrd[0], labtyp=xrd[1], marktyp=xrd[2], lti=xrd[3], tmod=xrd[4]
        var t = this.xtic_align_initial(step)
        //console.log("step "+step + " "+labtyp+" "+marktyp+" "+lti+" "+tmod+" => "+t)

        var redmark, lt, rlt, dt, label, w
        var tics = []

        this.C.font = this.opts.xtic_label_font
        var ht = this.C.measureText("m").width * 1.25

        for( ; t<this.xd_max; t += step ){

            redmark = 0
            if( t < this.xd_min ) continue
            lt  = this.localtime(t)
            rlt = this.localtime(t)
            // months go from 0. days from 1. absurd!
            lt.day --
            // mathematically, 28 is divisible by 7. but that just looks silly.
            if( lt.day > 22 && lti=='day' && tmod >= 7 ) lt.day = 22

	    if( step >= 3600*24 && lt.hour ){
	        // handle daylight saving time changes - resync to midnight
                dt = (lt.hour > 12 ? lt.hour - 24 : lt.hour) * 3600
                dt += ly.min * 60
                t -= dt + step
                continue
	    }
	    if( step >= 3600*24*31 && lt.day ){
	        // some months are not 31 days!
	        // also corrects years that do not leap
	        dt = lt.day * 3600*24
                t -= dt + step
                continue
	    }

            if( lt[lti] % tmod ) continue
            if( (lti == 'mon')  && (lt.day || lt.hour || lt.min || lt.sec )) continue
            if( (lti == 'day')  && (lt.hour || lt.min || lt.sec )) continue
            if( (lti == 'hour') && (lt.min || lt.sec )) continue
            if( (lti == 'min')  && lt.sec ) continue

            // if we are putting labels every 2 days,
            // do not place mark on 31st if we are going to put on on the 1st
            // it looks silly
            if( lt.day == 31 && tmod == 2 && lti == 'day' && (this.xd_max > t + 86400) ) continue

            if( marktyp == 'HR' && !lt.min ) redmark = 1 		// on the hour
            if( marktyp == 'MN' && !lt.hour && !lt.min ) redmark = 1 	// midnight
            if( marktyp == 'SU' && !lt.dow ) redmark = 1		// sunday
            if( marktyp == 'M1' && !lt.day ) redmark = 1		// 1st of month
            if( marktyp == 'Y1' && !lt.day && !lt.mon) redmark = 1	// 1 jan

            label = this.format_time(rlt, labtyp, redmark)
            w = this.C.measureText(label).width
            tics.push( {x: t, redmark: redmark, label: label, width: w, height: ht} )
        }

        if( this.opts.draw_tic_labels ){
            // move margin
            this.margin_bottom += ht + 5 + 4
        }

        this.xtics = tics

    }

    p.drawAxii = function(){
        var x = Math.round(this.px(0))
        var y = Math.round(this.py(0))

        this.C.beginPath()
        this.C.strokeStyle = this.opts.axii_color
        this.C.lineWidth = 1
        this.C.moveTo( x-.5, this.py(this.ymax) )
        this.C.lineTo( x-.5, y-.5 )
        this.C.lineTo( this.px(this.xmax), y-.5 )
        this.C.stroke()
    }

    p.drawGrid = function(){
        var i, x, y, tic
        var C = this.C

        C.save()
        C.lineWidth = 1

        if( !this.xtics || !this.ytics ) return
        
        for(i=0; i<this.ytics.length; i++){
            tic = this.ytics[i]
            y = Math.round(this.dy(tic.y))

            if( this.opts.draw_tics ){
                C.setLineDash([1,0])
                C.beginPath()
                C.strokeStyle = this.opts.ytic_color
                C.moveTo(this.px(-1),y+.5)
                C.lineTo(this.px(-4),y+.5)
                C.stroke()
            }
            if( this.opts.draw_grid ){
                C.setLineDash([1,3])
                C.beginPath()
                C.strokeStyle = this.opts.grid_color
                C.moveTo(this.px(0), y+.5)
                C.lineTo(this.dx(this.xd_max), y+.5)
                C.stroke()
            }
            if( this.opts.draw_tic_labels ){
                C.fillStyle = this.opts.ytic_label_color
                C.font = this.opts.ytic_label_font
                // almost, but not exactly centered
                C.fillText( tic.label, this.px(- tic.width)-6, y + tic.height/4)
            }
        }

        for( i=0; i<this.xtics.length; i++){
            tic = this.xtics[i]
            x = Math.round(this.dx(tic.x))

            if( this.opts.draw_tics ){
                C.setLineDash([1,0])
                C.beginPath()
                C.strokeStyle = this.opts.xtic_color
                C.moveTo(x+.5, this.py(-1))
                C.lineTo(x+.5, this.py(-4))
                C.stroke()
            }
            if( this.opts.draw_grid ){
                if( tic.redmark ){
                    C.setLineDash([0,0])
                }else{
                    C.setLineDash([1,3])
                }
                C.beginPath()
                C.strokeStyle = this.opts.grid_color
                C.moveTo(x+.5, this.py(0))
                C.lineTo(x+.5, this.dy(this.yd_max))
                C.stroke()
            }
            if( this.opts.draw_tic_labels ){
                if( tic.redmark ){
                    C.fillStyle = this.opts.mark_label_color
                }else{
                    C.fillStyle = this.opts.xtic_label_color
                }
                C.font = this.opts.xtic_label_font
                // almost, but not exactly centered
                C.fillText( tic.label, x - tic.width/3, this.py(0)+tic.height+5)
            }
        }

        C.restore()
    }


})()

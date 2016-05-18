package main

import (
	"encoding/json"
	"fmt"
	"log"
	"math"
	"net/http"
	"os"
	"strconv"
)

// Data structures
type Point2D struct {
	X float32 `json:"x"`
	Y float32 `json:"y"`
}

type Point3D struct {
	X float32 `json:"x"`
	Y float32 `json:"y"`
	Z float32 `json:"z"`
}

type ServerAnswer struct {
	ServerName string    `json:"servername"`
	Angle      int       `json:"angle"`
	Vertices   []Point2D `json:"vertices"`
	PointOrder [][]int   `json:"pointOrder"`
}

// Projection constants
var origVertices = []Point3D{
	{-1, 1, -1},
	{1, 1, -1},
	{1, -1, -1},
	{-1, -1, -1},
	{-1, 1, 1},
	{1, 1, 1},
	{1, -1, 1},
	{-1, -1, 1}}

const viewWidth = 400
const viewHeight = 200
const fov = 128
const viewDistance = 3.5

// The HTML page
const htmlContent = `
<!doctype html>
<html lang=en>
<head>
<meta charset=utf-8>
<title>rpi-cluster test app</title>
    <script type="text/javascript">
		// 200 ms by default
		var tempo = 200;

        window.onload = startDemo;

        var angle = 0;
		var appURI = window.location.protocol + "//" + window.location.host;

        var faces;
		var vertices2d;
		var intervalId;

		httpGetAsync(appURI + "/computeVertices?angle="+angle, function(json) {
			document.getElementById('serverAnswer').innerHTML = json
			obj = JSON.parse(json)
			angle = obj.angle
			faces = obj.pointOrder
            vertices2d = obj.vertices
		})

        function startDemo() {
			if (tempo < 10) {
				tempo = 10;
			}
			document.getElementById('speed').innerHTML = tempo

			if (intervalId) {
				clearInterval(intervalId);
			}
            canvas = document.getElementById("thecanvas");
            if( canvas && canvas.getContext ) {
                ctx = canvas.getContext("2d");
                intervalId = setInterval(loop,tempo);
            }
        }

        function loop() {
            var t = new Array();

            ctx.fillStyle = "rgb(0,0,0)";
            ctx.fillRect(0,0,400,200);

            ctx.strokeStyle = "rgb(255,55,255)"

            for( var i = 0; i < faces.length; i++ ) {
                var f = faces[i]
                ctx.beginPath()
                ctx.moveTo(vertices2d[f[0]].x,vertices2d[f[0]].y)
                ctx.lineTo(vertices2d[f[1]].x,vertices2d[f[1]].y)
                ctx.lineTo(vertices2d[f[2]].x,vertices2d[f[2]].y)
                ctx.lineTo(vertices2d[f[3]].x,vertices2d[f[3]].y)
                ctx.closePath()
                ctx.stroke()
            }

			angle += 2
			if (angle >= 360) {
				angle = 0;
			}
			httpGetAsync(appURI + "/computeVertices?angle="+angle, function(json) {
				document.getElementById('serverAnswer').innerHTML = json
			    obj = JSON.parse(json)
			    angle = obj.angle
			    faces = obj.pointOrder
                vertices2d = obj.vertices
			})
        }

		function httpGetAsync(theUrl, callback) {
            var xmlHttp = new XMLHttpRequest();
	        xmlHttp.onreadystatechange = function() {
		    if (xmlHttp.readyState == 4 && xmlHttp.status == 200)
			    callback(xmlHttp.responseText);
			}
			xmlHttp.open("GET", theUrl, true); // true for asynchronous
			xmlHttp.send(null);
		}
    </script>
</head>
<body>
    <canvas id="thecanvas" width="400" height="200">
       Your browser does not support the HTML5 canvas element.
    </canvas>
<p>
Wait between requests : <span id="speed"></span> ms
<button type="button" onclick="tempo += 10; startDemo();">Slower</button>&nbsp;
<button type="button" onclick="tempo -= 10; startDemo();">Faster</button>&nbsp;
</p>

<h1>Response from server : </h1>
<p id="serverAnswer"></p>
</body>
</html>
`

// Rotation functions
func rotateX(p3d Point3D, angle int) Point3D {
	rad := float64(angle) * math.Pi / 180
	cosa := float32(math.Cos(rad))
	sina := float32(math.Sin(rad))
	y := p3d.Y*cosa - p3d.Z*sina
	z := p3d.Y*sina + p3d.Z*cosa
	return Point3D{p3d.X, y, z}
}

func rotateY(p3d Point3D, angle int) Point3D {
	rad := float64(angle) * math.Pi / 180
	cosa := float32(math.Cos(rad))
	sina := float32(math.Sin(rad))
	z := p3d.Z*cosa - p3d.X*sina
	x := p3d.Z*sina + p3d.X*cosa
	return Point3D{x, p3d.Y, z}
}

func rotateZ(p3d Point3D, angle int) Point3D {
	rad := float64(angle) * math.Pi / 180
	cosa := float32(math.Cos(rad))
	sina := float32(math.Sin(rad))
	x := p3d.X*cosa - p3d.Y*sina
	y := p3d.X*sina + p3d.Y*cosa
	return Point3D{x, y, p3d.Z}
}

// Projection function
func project(p3d Point3D) Point2D {
	factor := fov / (viewDistance + p3d.Z)
	x := p3d.X*factor + viewWidth/2
	y := p3d.Y*factor + viewHeight/2

	return Point2D{x, y}
}

// ==== Main Process ====
func main() {
	name, err := os.Hostname()
	if err != nil {
		panic(err)
	}

	envName := os.Getenv("CUBEHOST")

	var serverAnswer ServerAnswer

	if envName != "" {
		serverAnswer.ServerName = envName
	} else {
		serverAnswer.ServerName = name
	}
	serverAnswer.Angle = 0
	serverAnswer.PointOrder = [][]int{{0, 1, 2, 3}, {1, 5, 6, 2}, {5, 4, 7, 6}, {4, 0, 3, 7}, {0, 4, 5, 1}, {3, 2, 6, 7}}

	// HTML page entry point
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, htmlContent)
	})

	// HTML page entry point
	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "Health : OK")
	})

	// REST API entry point
	http.HandleFunc("/computeVertices", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json; charset=UTF-8")
		w.WriteHeader(http.StatusOK)
		angleQuery := r.URL.Query().Get("angle")
		angle, err := strconv.Atoi(angleQuery)
		if err != nil {
			panic(err)
		}

		serverAnswer.Vertices = make([]Point2D, len(origVertices))
		for index, p3d := range origVertices {
			rotP3d := rotateZ(rotateY(rotateX(p3d, angle), angle), angle)
			serverAnswer.Vertices[index] = project(rotP3d)
		}

		serverAnswer.Angle = angle
		if err := json.NewEncoder(w).Encode(serverAnswer); err != nil {
			panic(err)
		}
	})

	fmt.Println("Service running on port 8081")
	fmt.Println("Type [CTRL]+[C] to quit!")

	log.Fatal(http.ListenAndServe(":8081", nil))
}

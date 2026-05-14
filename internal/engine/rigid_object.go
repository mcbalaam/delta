package engine

import (
	"image/color"
	"math"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
	"github.com/mcbalaam/delta/internal/render"
)

type Vec struct{ X, Y float64 }

func transformPoint(p Vec, sx, sy, rot, tx, ty float64) Vec {
	x, y := p.X*sx, p.Y*sy
	c, s := math.Cos(rot), math.Sin(rot)
	rx := x*c - y*s
	ry := x*s + y*c
	return Vec{rx + tx, ry + ty}
}

func projectPoly(axis Vec, verts []Vec) (min, max float64) {
	min, max = math.Inf(1), math.Inf(-1)
	for _, v := range verts {
		proj := v.X*axis.X + v.Y*axis.Y
		if proj < min {
			min = proj
		}
		if proj > max {
			max = proj
		}
	}
	return
}

func overlapOnAxis(aMin, aMax, bMin, bMax float64) bool {
	return !(aMax < bMin || bMax < aMin)
}

func polygonsIntersect(a, b []Vec) bool {
	// iterate edges of both polygons
	checkAxes := func(verts []Vec) bool {
		n := len(verts)
		for i := 0; i < n; i++ {
			j := (i + 1) % n

			ex := verts[j].X - verts[i].X
			ey := verts[j].Y - verts[i].Y

			axis := Vec{-ey, ex}

			aMin, aMax := projectPoly(axis, a)
			bMin, bMax := projectPoly(axis, b)
			if !overlapOnAxis(aMin, aMax, bMin, bMax) {
				return false
			}
		}
		return true
	}

	return checkAxes(a) && checkAxes(b)
}

func RectToPoly(width, height float64) []Vec {
	return []Vec{
		{0, 0},
		{width, 0},
		{width, height},
		{0, height},
	}
}

func TransformPoly(verts []Vec, sx, sy, rot, tx, ty float64) []Vec {
	out := make([]Vec, len(verts))
	for i, v := range verts {
		out[i] = transformPoint(v, sx, sy, rot, tx, ty)
	}
	return out
}

type Hitbox struct {
	LocalVerts []Vec
	WorldVerts []Vec
	Width      float64
	Height     float64
	OffsetX    float64
	OffsetY    float64
	Rotation   float64
}

// RigidObject is an Object with a dynamic hitbox
type RigidObject struct {
	Object

	Hitbox *Hitbox
}

func (r *RigidObject) DrawHitboxDebug(s *ebiten.Image) {
	if r.Hitbox == nil || len(r.Hitbox.WorldVerts) < 2 {
		return
	}

	for i := 0; i < len(r.Hitbox.WorldVerts); i++ {
		current := r.Hitbox.WorldVerts[i]
		next := r.Hitbox.WorldVerts[(i+1)%len(r.Hitbox.WorldVerts)]

		vector.StrokeLine(s, float32(current.X), float32(current.Y),
			float32(next.X), float32(next.Y), 2, color.RGBA{0, 255, 0, 200}, true)
	}
}

func NewRigidObject(posx, posy, velx, vely, scalex, scaley, rotation float64, icon render.AnimatedIcon, width, height, xoffset, yoffset float64) *RigidObject {
	halfW := width / 2
	halfH := height / 2

	obj := &RigidObject{
		Object: *NewObject(posx, posy, velx, vely, scalex, scaley, rotation, icon),
		Hitbox: &Hitbox{
			LocalVerts: []Vec{
				{-halfW, -halfH},
				{halfW, -halfH},
				{halfW, halfH},
				{-halfW, halfH},
			},
			WorldVerts: make([]Vec, 4),
			Width:      width,
			Height:     height,
			OffsetX:    xoffset,
			OffsetY:    yoffset,
			Rotation:   0,
		},
	}
	obj.UpdateHitbox()
	DefaultQueue.Schedule(obj)
	DefaultUpdateQueue.Schedule(obj)
	return obj
}

func (obj *RigidObject) SetHitboxOffset(ox, oy float64) {
	obj.Hitbox.OffsetX = ox
	obj.Hitbox.OffsetY = oy
}

func (obj *RigidObject) CenterHitbox() {
	obj.SetHitboxOffset(
		-obj.Hitbox.Width/2,
		-obj.Hitbox.Height/2,
	)
}

func (r *RigidObject) Draw(s *ebiten.Image) {
	offsetVec := Vec{r.Hitbox.OffsetX, r.Hitbox.OffsetY}
	transformedOffset := transformPoint(
		offsetVec,
		r.ScaleX, r.ScaleY,
		r.Rotation,
		0, 0,
	)

	centerX := r.PosX - (r.Hitbox.Width * r.ScaleX / 2)
	centerY := r.PosY - (r.Hitbox.Height * r.ScaleY / 2)

	iconX := centerX + transformedOffset.X
	iconY := centerY + transformedOffset.Y

	r.Icon.Draw(s, iconX, iconY, r.ScaleX, r.ScaleY, r.Rotation)
	r.DrawHitboxDebug(s)
}

func (r *RigidObject) UpdateHitbox() {
	if r.Hitbox == nil || len(r.Hitbox.LocalVerts) == 0 {
		return
	}

	for i, localVert := range r.Hitbox.LocalVerts {
		r.Hitbox.WorldVerts[i] = transformPoint(
			localVert,
			r.ScaleX, r.ScaleY,
			r.Rotation,
			r.PosX,
			r.PosY,
		)
	}
}

func (obj *RigidObject) Move(dx, dy float64) {
	obj.PosX += dx
	obj.PosY += dy
	obj.UpdateHitbox()
}

func (obj *RigidObject) SetPosition(x, y float64) {
	obj.PosX = x
	obj.PosY = y
	obj.UpdateHitbox()
}

func (obj *RigidObject) Rotate(angle float64) {
	obj.Rotation += angle
	obj.UpdateHitbox()
}

func (obj *RigidObject) SetRotation(angle float64) {
	obj.Rotation = angle
	obj.UpdateHitbox()
}

func (obj *RigidObject) SetScale(scale float64) {
	obj.ScaleX = scale
	obj.ScaleY = scale
	obj.UpdateHitbox()
}

func (r *RigidObject) Update(dt time.Duration) {
	r.Object.Update(dt)
	r.UpdateHitbox()
}

func (obj *RigidObject) CollidesWith(other *RigidObject) bool {
	if obj.Hitbox == nil || other.Hitbox == nil {
		return false
	}
	return polygonsIntersect(obj.Hitbox.WorldVerts, other.Hitbox.WorldVerts)
}

func (obj *RigidObject) GetBounds() (minX, minY, maxX, maxY float64) {
	if obj.Hitbox == nil || len(obj.Hitbox.WorldVerts) == 0 {
		return obj.PosX, obj.PosY, obj.PosX, obj.PosY
	}

	minX = math.Inf(1)
	minY = math.Inf(1)
	maxX = math.Inf(-1)
	maxY = math.Inf(-1)

	for _, v := range obj.Hitbox.WorldVerts {
		if v.X < minX {
			minX = v.X
		}
		if v.Y < minY {
			minY = v.Y
		}
		if v.X > maxX {
			maxX = v.X
		}
		if v.Y > maxY {
			maxY = v.Y
		}
	}

	return
}

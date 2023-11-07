package main

import (
	"embed"
	"fmt"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/lafriks/go-tiled"
	"github.com/solarlune/paths"
	"log"
	"math"
	"path"
	"strings"
)

//go:embed assets/*
var embeddedFiles embed.FS

type PathMapDemo struct {
	Level          *tiled.Map
	tileHash       map[uint32]*ebiten.Image
	pathFindingMap []string
	coins          coinPile
	npc            NonPlayerChar
	pathMap        *paths.Grid
	path           *paths.Path
}

type coinPile struct {
	pict   *ebiten.Image
	row    int
	column int
}

type NonPlayerChar struct {
	pict *ebiten.Image
	xloc float64
	yloc float64
}

func (demo *PathMapDemo) Update() error {
	checkMouse(demo)
	if demo.path != nil {
		pathCell := demo.path.Current()
		if math.Abs(float64(pathCell.X*demo.Level.TileWidth)-(demo.npc.xloc)) <= 2 &&
			math.Abs(float64(pathCell.Y*demo.Level.TileHeight)-(demo.npc.yloc)) <= 2 { //if we are now on the tile we need to be on
			demo.path.Advance()
		}
		direction := 0.0
		if pathCell.X*demo.Level.TileWidth > int(demo.npc.xloc) {
			direction = 1.0
		} else if pathCell.X*demo.Level.TileWidth < int(demo.npc.xloc) {
			direction = -1.0
		}
		Ydirection := 0.0
		if pathCell.Y*demo.Level.TileHeight > int(demo.npc.yloc) {
			Ydirection = 1.0
		} else if pathCell.Y*demo.Level.TileHeight < int(demo.npc.yloc) {
			Ydirection = -1.0
		}
		demo.npc.xloc += direction * 2
		demo.npc.yloc += Ydirection * 2
	}
	return nil
}

func checkMouse(demo *PathMapDemo) {
	if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
		mouseX, mouseY := ebiten.CursorPosition()
		demo.npc.xloc = float64(mouseX)
		demo.npc.yloc = float64(mouseY)
		startRow := int(demo.npc.yloc) / demo.Level.TileHeight
		startCol := int(demo.npc.xloc) / demo.Level.TileWidth
		startCell := demo.pathMap.Get(startCol, startRow)
		endCell := demo.pathMap.Get(demo.coins.column, demo.coins.row)
		demo.path = demo.pathMap.GetPathFromCells(startCell, endCell, false, false)
	}
}

func (demo PathMapDemo) Draw(screen *ebiten.Image) {
	drawOptions := ebiten.DrawImageOptions{}
	//draw map
	for tileY := 0; tileY < demo.Level.Height; tileY += 1 {
		for tileX := 0; tileX < demo.Level.Width; tileX += 1 {
			drawOptions.GeoM.Reset()
			TileXpos := float64(demo.Level.TileWidth * tileX)
			TileYpos := float64(demo.Level.TileHeight * tileY)
			drawOptions.GeoM.Translate(TileXpos, TileYpos)
			tileToDraw := demo.Level.Layers[0].Tiles[tileY*demo.Level.Width+tileX]
			ebitenTileToDraw := demo.tileHash[tileToDraw.ID]
			screen.DrawImage(ebitenTileToDraw, &drawOptions)
		}
	}
	//draw gold
	drawOptions.GeoM.Reset()
	drawOptions.GeoM.Translate(float64(demo.coins.column*demo.Level.TileWidth), float64(demo.coins.row*demo.Level.TileHeight))
	screen.DrawImage(demo.coins.pict, &drawOptions)
	//draw goblin
	drawOptions.GeoM.Reset()
	drawOptions.GeoM.Translate(demo.npc.xloc, demo.npc.yloc)
	screen.DrawImage(demo.npc.pict, &drawOptions)
}

func (demo PathMapDemo) Layout(outsideWidth, outsideHeight int) (screenWidth, screenHeight int) {
	return outsideWidth, outsideHeight
}

func main() {
	gameMap := loadMapFromEmbedded(path.Join("assets", "MapForPaths.tmx"))
	pathMap := makeSearchMap(gameMap)
	searchablePathMap := paths.NewGridFromStringArrays(pathMap, gameMap.TileWidth, gameMap.TileHeight)
	searchablePathMap.SetWalkable('2', false)
	searchablePathMap.SetWalkable('3', false)
	coins := makeCoinPile()
	nonPlayer := makeNPC()
	ebiten.SetWindowSize(gameMap.TileWidth*gameMap.Width, gameMap.TileHeight*gameMap.Height)
	ebiten.SetWindowTitle("Maps Embedded")
	ebitenImageMap := makeEbitenImagesFromMap(*gameMap)
	oneLevelGame := PathMapDemo{
		Level:          gameMap,
		tileHash:       ebitenImageMap,
		pathFindingMap: pathMap,
		coins:          coins,
		npc:            nonPlayer,
		pathMap:        searchablePathMap,
	}
	err := ebiten.RunGame(&oneLevelGame)
	if err != nil {
		fmt.Println("Couldn't run game:", err)
	}
}

func makeNPC() NonPlayerChar {
	picture := LoadEmbeddedImage("", "goblin.png")
	character := NonPlayerChar{
		pict: picture,
		xloc: -100, //put the NPC off screen originally
		yloc: -100,
	}
	return character
}

func makeCoinPile() coinPile {
	picture := LoadEmbeddedImage("", "coins.png")
	money := coinPile{
		pict:   picture,
		row:    12,
		column: 10,
	}
	return money
}

func loadMapFromEmbedded(name string) *tiled.Map {
	embeddedMap, err := tiled.LoadFile(name, tiled.WithFileSystem(embeddedFiles))
	if err != nil {
		fmt.Println("Error loading embedded map:", err)
	}
	return embeddedMap
}

func makeSearchMap(tiledMap *tiled.Map) []string {
	mapAsStringSlice := make([]string, 0, tiledMap.Height) //each row will be its own string
	row := strings.Builder{}
	for position, tile := range tiledMap.Layers[0].Tiles {
		if position%tiledMap.Width == 0 && position > 0 { // we get the 2d array as an unrolled one-d array
			mapAsStringSlice = append(mapAsStringSlice, row.String())
			row = strings.Builder{}
		}
		row.WriteString(fmt.Sprintf("%d", tile.ID))
	}
	mapAsStringSlice = append(mapAsStringSlice, row.String())
	return mapAsStringSlice
}

func makeEbitenImagesFromMap(tiledMap tiled.Map) map[uint32]*ebiten.Image {
	idToImage := make(map[uint32]*ebiten.Image)
	for _, tile := range tiledMap.Tilesets[0].Tiles {
		embeddedFile, err := embeddedFiles.Open(path.Join("assets", tile.Image.Source))
		if err != nil {
			log.Fatal("failed to load embedded image ", embeddedFile, err)
		}
		ebitenImageTile, _, err := ebitenutil.NewImageFromReader(embeddedFile)
		if err != nil {
			fmt.Println("Error loading tile image:", tile.Image.Source, err)
		}
		idToImage[tile.ID] = ebitenImageTile
	}
	return idToImage
}

func LoadEmbeddedImage(folderName string, imageName string) *ebiten.Image {
	embeddedFile, err := embeddedFiles.Open(path.Join("assets", folderName, imageName))
	if err != nil {
		log.Fatal("failed to load embedded image ", imageName, err)
	}
	ebitenImage, _, err := ebitenutil.NewImageFromReader(embeddedFile)
	if err != nil {
		fmt.Println("Error loading tile image:", imageName, err)
	}
	return ebitenImage
}

package gui

import (
	"fmt"
	"math/rand"
	"sync"
	"sync/atomic"

	"fyne.io/fyne/v2"

	"github.com/ikari-pl/go-cpc-image/pkg/bitmap"
	"github.com/ikari-pl/go-cpc-image/pkg/convert"
	"github.com/ikari-pl/go-cpc-image/pkg/render"
)

// evalSettings runs a conversion with the given settings and returns the MSE
// compared to the reference source image (already at CPC resolution).
func (app *Application) evalSettings(source *bitmap.DirectBitmap, prm *convert.Settings) float64 {
	cpcW := prm.GetScreenWidth()
	cpcH := prm.GetScreenHeight()

	// Convert modifies the source bitmap in-place, so work on a copy.
	srcCopy := bitmap.NewDirectBitmap(source.Width(), source.Height())
	srcCopy.CopyBits(source)

	dest := &convert.ImageCpc{
		BitmapCpc:  render.NewBitmapCpcWithParams(prm.NumCols, prm.NumLines, prm.CpcPlus),
		DisplayBmp: bitmap.NewDirectBitmap(cpcW, cpcH),
		Width:      cpcW,
		Height:     cpcH,
	}

	convert.Convert(srcCopy, dest, prm, true)
	return convert.ComputeMSE(source, dest.DisplayBmp)
}

// AutoOptimize tries many parameter combinations and applies the best-scoring settings.
// Runs a full greedy pass for each of modes 0, 1, 2 (~65×3 = ~195 conversions).
// Must be called from a goroutine.
func (app *Application) AutoOptimize() {
	update, dismiss := app.ShowProgress("Auto-optimize")
	defer dismiss()

	source := app.getResizeBitmap()
	if source == nil {
		app.SetStatus("Auto-optimize: no source image")
		return
	}

	step := 0
	total := 65 * 3 // ~65 per mode × 3 modes
	report := func() {
		step++
		update(float64(step)/float64(total), fmt.Sprintf("Testing %d/%d...", step, total))
	}

	base := *app.params
	var best convert.Settings
	bestScore := 1e18

	// Run full greedy sweep for each mode independently
	for _, mode := range []int{0, 1, 2} {
		candidate := base
		candidate.VirtualMode = mode
		result, score := app.greedyOptimizeWithReport(source, candidate, report)
		if score < bestScore {
			bestScore = score
			best = result
		}
	}

	fyne.Do(func() {
		*app.params = best
		app.RefreshControls()
		app.TriggerConversion()
	})

	app.SetStatus(fmt.Sprintf("Auto-optimize: done (MSE: %.1f, mode %d)", bestScore, best.VirtualMode))
}

// genome represents one individual in the genetic algorithm population.
type genome struct {
	settings convert.Settings
	score    float64
}

// randomGenome creates a random settings variant based on a template.
// Fixed fields (VirtualMode, CpcPlus, NumCols, NumLines, palette locks, crop, RGB pcts) are preserved.
func randomGenome(base *convert.Settings, methods []string, rng *rand.Rand) convert.Settings {
	s := *base

	// Boolean flags — random
	s.Smoothing = rng.Intn(2) == 1
	s.TrueColorDither = rng.Intn(2) == 1
	s.NewMethod = rng.Intn(2) == 1
	s.NewReduc = rng.Intn(2) == 1
	s.ReductPal1 = rng.Intn(2) == 1
	s.ReductPal2 = rng.Intn(2) == 1
	s.ReductPal3 = rng.Intn(2) == 1
	s.ReductPal4 = rng.Intn(2) == 1
	s.DiffErr = rng.Intn(2) == 1
	s.Filter = rng.Intn(2) == 1

	// Dithering method
	s.Method = methods[rng.Intn(len(methods))]

	// Dithering percentage 0-100
	s.Pct = rng.Intn(101)

	// Bit depth
	depths := []int{6, 9, 12, 24}
	s.BitsRGB = depths[rng.Intn(len(depths))]

	// Colour adjustments 75-125 (narrower range — extremes are almost never good)
	s.PctLumi = 75 + rng.Intn(51)
	s.PctSat = 75 + rng.Intn(51)
	s.PctContrast = 75 + rng.Intn(51)

	return s
}

// crossover produces a child by mixing two parents' genes randomly.
func crossover(a, b *convert.Settings, methods []string, rng *rand.Rand) convert.Settings {
	pick := func() bool { return rng.Intn(2) == 0 }
	pickBool := func(va, vb bool) bool {
		if pick() {
			return va
		}
		return vb
	}
	pickInt := func(va, vb int) int {
		if pick() {
			return va
		}
		return vb
	}

	child := *a // start from a to inherit fixed fields

	child.Smoothing = pickBool(a.Smoothing, b.Smoothing)
	child.TrueColorDither = pickBool(a.TrueColorDither, b.TrueColorDither)
	child.NewMethod = pickBool(a.NewMethod, b.NewMethod)
	child.NewReduc = pickBool(a.NewReduc, b.NewReduc)
	child.ReductPal1 = pickBool(a.ReductPal1, b.ReductPal1)
	child.ReductPal2 = pickBool(a.ReductPal2, b.ReductPal2)
	child.ReductPal3 = pickBool(a.ReductPal3, b.ReductPal3)
	child.ReductPal4 = pickBool(a.ReductPal4, b.ReductPal4)
	child.DiffErr = pickBool(a.DiffErr, b.DiffErr)
	child.Filter = pickBool(a.Filter, b.Filter)

	if pick() {
		child.Method = b.Method
	}
	child.Pct = pickInt(a.Pct, b.Pct)
	child.BitsRGB = pickInt(a.BitsRGB, b.BitsRGB)
	child.PctLumi = pickInt(a.PctLumi, b.PctLumi)
	child.PctSat = pickInt(a.PctSat, b.PctSat)
	child.PctContrast = pickInt(a.PctContrast, b.PctContrast)

	return child
}

// mutate randomly perturbs a genome's mutable fields.
func mutate(s *convert.Settings, methods []string, rng *rand.Rand) {
	// Flip a random boolean (1 in 3 chance per call)
	if rng.Intn(3) == 0 {
		bools := []*bool{
			&s.Smoothing, &s.TrueColorDither, &s.NewMethod, &s.NewReduc,
			&s.ReductPal1, &s.ReductPal2, &s.ReductPal3, &s.ReductPal4,
			&s.DiffErr, &s.Filter,
		}
		b := bools[rng.Intn(len(bools))]
		*b = !*b
	}

	// Change dithering method (1 in 4)
	if rng.Intn(4) == 0 {
		s.Method = methods[rng.Intn(len(methods))]
	}

	// Nudge dithering percentage (1 in 3)
	if rng.Intn(3) == 0 {
		s.Pct = clampInt(s.Pct+rng.Intn(21)-10, 0, 100)
	}

	// Nudge colour adjustments (1 in 3 each)
	if rng.Intn(3) == 0 {
		s.PctLumi = clampInt(s.PctLumi+rng.Intn(21)-10, 50, 150)
	}
	if rng.Intn(3) == 0 {
		s.PctSat = clampInt(s.PctSat+rng.Intn(21)-10, 50, 150)
	}
	if rng.Intn(3) == 0 {
		s.PctContrast = clampInt(s.PctContrast+rng.Intn(21)-10, 50, 150)
	}

	// Change bit depth (1 in 5)
	if rng.Intn(5) == 0 {
		depths := []int{6, 9, 12, 24}
		s.BitsRGB = depths[rng.Intn(len(depths))]
	}
}

func clampInt(v, lo, hi int) int {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

// tournamentSelect picks the best of 3 random individuals.
func tournamentSelect(pop []genome, rng *rand.Rand) *genome {
	best := &pop[rng.Intn(len(pop))]
	for i := 0; i < 2; i++ {
		c := &pop[rng.Intn(len(pop))]
		if c.score < best.score {
			best = c
		}
	}
	return best
}

// greedyOptimize runs greedy hill-climbing without progress reporting.
func (app *Application) greedyOptimize(source *bitmap.DirectBitmap, base convert.Settings) (convert.Settings, float64) {
	return app.greedyOptimizeWithReport(source, base, func() {})
}

// greedyOptimizeWithReport runs greedy hill-climbing, calling report after each eval.
func (app *Application) greedyOptimizeWithReport(source *bitmap.DirectBitmap, base convert.Settings, report func()) (convert.Settings, float64) {
	best := base
	bestScore := app.evalSettings(source, &best)
	report()

	tryBetter := func(candidate *convert.Settings) {
		score := app.evalSettings(source, candidate)
		report()
		if score < bestScore {
			bestScore = score
			best = *candidate
		}
	}

	// Boolean flags
	for _, toggle := range []func(*convert.Settings){
		func(s *convert.Settings) { s.Smoothing = !s.Smoothing },
		func(s *convert.Settings) { s.TrueColorDither = !s.TrueColorDither },
		func(s *convert.Settings) { s.NewMethod = !s.NewMethod },
		func(s *convert.Settings) { s.NewReduc = !s.NewReduc },
		func(s *convert.Settings) { s.ReductPal1 = !s.ReductPal1 },
		func(s *convert.Settings) { s.ReductPal2 = !s.ReductPal2 },
		func(s *convert.Settings) { s.ReductPal3 = !s.ReductPal3 },
		func(s *convert.Settings) { s.ReductPal4 = !s.ReductPal4 },
		func(s *convert.Settings) { s.DiffErr = !s.DiffErr },
		func(s *convert.Settings) { s.Filter = !s.Filter },
	} {
		c := best
		toggle(&c)
		tryBetter(&c)
	}

	for _, m := range convert.GetAvailableMethods() {
		c := best
		c.Method = m
		tryBetter(&c)
	}
	for pct := 0; pct <= 100; pct += 10 {
		c := best
		c.Pct = pct
		tryBetter(&c)
	}
	for _, bits := range []int{6, 9, 12, 24} {
		c := best
		c.BitsRGB = bits
		tryBetter(&c)
	}
	for _, lumi := range []int{80, 90, 100, 110, 120} {
		c := best
		c.PctLumi = lumi
		tryBetter(&c)
	}
	for _, sat := range []int{80, 90, 100, 110, 120} {
		c := best
		c.PctSat = sat
		tryBetter(&c)
	}
	for _, ct := range []int{80, 90, 100, 110, 120} {
		c := best
		c.PctContrast = ct
		tryBetter(&c)
	}
	return best, bestScore
}

// DeepOptimize first runs a greedy pass (like Auto) to find a good seed,
// then runs a genetic algorithm with parallel workers to refine further.
// Population of 40, runs 30 generations. Must be called from a goroutine.
func (app *Application) DeepOptimize() {
	update, dismiss := app.ShowProgress("Deep optimize")
	defer dismiss()

	source := app.getResizeBitmap()
	if source == nil {
		app.SetStatus("Deep optimize: no source image")
		return
	}

	base := *app.params
	methods := convert.GetAvailableMethods()
	var evalCount atomic.Int64

	// Per-mode GA parameters
	const popSize = 30
	const generations = 20
	const numWorkers = 4
	evalsPerMode := popSize + (popSize-1)*generations // ~600
	totalEvals := evalsPerMode*3 + 65*3              // GA + greedy seeds for 3 modes

	var bestSettings convert.Settings
	bestScore := 1e18

	for _, mode := range []int{0, 1, 2} {
		modeBase := base
		modeBase.VirtualMode = mode

		// Greedy seed for this mode
		update(float64(evalCount.Load())/float64(totalEvals),
			fmt.Sprintf("Mode %d: greedy seed...", mode))
		greedySeed, greedyScore := app.greedyOptimize(source, modeBase)
		evalCount.Add(65)

		eval := func(prm *convert.Settings) float64 {
			score := app.evalSettings(source, prm)
			n := evalCount.Add(1)
			update(float64(n)/float64(totalEvals),
				fmt.Sprintf("Mode %d: %d/%d evals...", mode, n, totalEvals))
			return score
		}

		rng := rand.New(rand.NewSource(rand.Int63()))
		pop := make([]genome, popSize)

		pop[0].settings = greedySeed
		pop[0].score = greedyScore

		// Light mutations of greedy seed
		for i := 1; i < 8 && i < popSize; i++ {
			s := greedySeed
			mutate(&s, methods, rng)
			pop[i].settings = s
		}
		// Random genomes (same mode)
		for i := 8; i < popSize; i++ {
			pop[i].settings = randomGenome(&modeBase, methods, rng)
		}

		evalPop := func(toEval []int) {
			ch := make(chan int, len(toEval))
			for _, idx := range toEval {
				ch <- idx
			}
			close(ch)
			var wg sync.WaitGroup
			for w := 0; w < numWorkers; w++ {
				wg.Add(1)
				go func() {
					defer wg.Done()
					for idx := range ch {
						pop[idx].score = eval(&pop[idx].settings)
					}
				}()
			}
			wg.Wait()
		}

		// Evaluate initial population (skip slot 0)
		initIdxs := make([]int, popSize-1)
		for i := range initIdxs {
			initIdxs[i] = i + 1
		}
		evalPop(initIdxs)

		for gen := 0; gen < generations; gen++ {
			next := make([]genome, popSize)
			bestIdx := 0
			for i := 1; i < popSize; i++ {
				if pop[i].score < pop[bestIdx].score {
					bestIdx = i
				}
			}
			next[0] = pop[bestIdx]

			for i := 1; i < popSize; i++ {
				p1 := tournamentSelect(pop, rng)
				p2 := tournamentSelect(pop, rng)
				child := crossover(&p1.settings, &p2.settings, methods, rng)
				mutate(&child, methods, rng)
				next[i].settings = child
			}
			pop = next

			newIdxs := make([]int, popSize-1)
			for i := range newIdxs {
				newIdxs[i] = i + 1
			}
			evalPop(newIdxs)
		}

		// Best from this mode's GA
		modeBestIdx := 0
		for i := 1; i < popSize; i++ {
			if pop[i].score < pop[modeBestIdx].score {
				modeBestIdx = i
			}
		}
		if pop[modeBestIdx].score < bestScore {
			bestScore = pop[modeBestIdx].score
			bestSettings = pop[modeBestIdx].settings
		}
	}

	fyne.Do(func() {
		*app.params = bestSettings
		app.RefreshControls()
		app.TriggerConversion()
	})

	app.SetStatus(fmt.Sprintf("Deep optimize: done (MSE: %.1f, mode %d, %d evals)",
		bestScore, bestSettings.VirtualMode, evalCount.Load()))
}

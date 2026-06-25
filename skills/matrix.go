package skills

import (
	"fmt"
	"math"
	"strings"
)

type MatrixConfig struct {
	Operation string      `json:"operation"`
	MatrixA   [][]float64 `json:"matrix_a"`
	MatrixB   [][]float64 `json:"matrix_b,omitempty"`
	Scalar    float64     `json:"scalar,omitempty"`
}

type MatrixResult struct {
	Output     string
	LatexExprs []LatexExpr
}

func RunMatrix(cfg MatrixConfig) (MatrixResult, error) {
	A := cfg.MatrixA
	if len(A) == 0 {
		return MatrixResult{}, fmt.Errorf("matrix_a is required")
	}
	if len(A) > 6 || len(A[0]) > 6 {
		return MatrixResult{}, fmt.Errorf("max matrix size is 6×6")
	}

	result := MatrixResult{}

	switch cfg.Operation {
	case "determinant":
		if len(A) != len(A[0]) {
			return MatrixResult{}, fmt.Errorf("determinant requires a square matrix")
		}
		det := determinant(A)
		result.Output = fmt.Sprintf("det(A) = **%.6g**", det)
		result.LatexExprs = []LatexExpr{
			{Label: "A", Expr: matrixToLatex(A)},
			{Label: "det(A)", Expr: fmt.Sprintf("%.6g", det)},
		}

	case "transpose":
		T := transpose(A)
		result.Output = fmt.Sprintf("transpose:\n%s", matrixToString(T))
		result.LatexExprs = []LatexExpr{
			{Label: "A", Expr: matrixToLatex(A)},
			{Label: "A^T", Expr: matrixToLatex(T)},
		}

	case "inverse":
		if len(A) != len(A[0]) {
			return MatrixResult{}, fmt.Errorf("inverse requires a square matrix")
		}
		inv, err := inverse(A)
		if err != nil {
			return MatrixResult{}, err
		}
		result.Output = fmt.Sprintf("A⁻¹:\n%s", matrixToString(inv))
		result.LatexExprs = []LatexExpr{
			{Label: "A", Expr: matrixToLatex(A)},
			{Label: "A^{-1}", Expr: matrixToLatex(inv)},
		}

	case "multiply":
		B := cfg.MatrixB
		if len(B) == 0 {
			return MatrixResult{}, fmt.Errorf("matrix_b required for multiply")
		}
		if len(A[0]) != len(B) {
			return MatrixResult{}, fmt.Errorf("incompatible dimensions: A is %dx%d, B is %dx%d", len(A), len(A[0]), len(B), len(B[0]))
		}
		C := multiply(A, B)
		result.Output = fmt.Sprintf("A × B:\n%s", matrixToString(C))
		result.LatexExprs = []LatexExpr{
			{Label: "A", Expr: matrixToLatex(A)},
			{Label: "B", Expr: matrixToLatex(B)},
			{Label: "A \\times B", Expr: matrixToLatex(C)},
		}

	case "add":
		B := cfg.MatrixB
		if len(B) == 0 {
			return MatrixResult{}, fmt.Errorf("matrix_b required for add")
		}
		if len(A) != len(B) || len(A[0]) != len(B[0]) {
			return MatrixResult{}, fmt.Errorf("matrices must have same dimensions")
		}
		C := addMatrix(A, B)
		result.Output = fmt.Sprintf("A + B:\n%s", matrixToString(C))
		result.LatexExprs = []LatexExpr{
			{Label: "A", Expr: matrixToLatex(A)},
			{Label: "B", Expr: matrixToLatex(B)},
			{Label: "A + B", Expr: matrixToLatex(C)},
		}

	case "scalar_multiply":
		C := scalarMultiply(A, cfg.Scalar)
		result.Output = fmt.Sprintf("%.6g × A:\n%s", cfg.Scalar, matrixToString(C))
		result.LatexExprs = []LatexExpr{
			{Label: "A", Expr: matrixToLatex(A)},
			{Label: fmt.Sprintf("%.6g A", cfg.Scalar), Expr: matrixToLatex(C)},
		}

	case "trace":
		if len(A) != len(A[0]) {
			return MatrixResult{}, fmt.Errorf("trace requires a square matrix")
		}
		tr := trace(A)
		result.Output = fmt.Sprintf("tr(A) = **%.6g**", tr)
		result.LatexExprs = []LatexExpr{
			{Label: "A", Expr: matrixToLatex(A)},
			{Label: "tr(A)", Expr: fmt.Sprintf("%.6g", tr)},
		}

	case "rank":
		r := rank(A)
		result.Output = fmt.Sprintf("rank(A) = **%d**", r)
		result.LatexExprs = []LatexExpr{
			{Label: "A", Expr: matrixToLatex(A)},
		}

	default:
		return MatrixResult{}, fmt.Errorf("unknown operation %q. supported: determinant, transpose, inverse, multiply, add, scalar_multiply, trace, rank", cfg.Operation)
	}

	return result, nil
}

func determinant(m [][]float64) float64 {
	n := len(m)
	if n == 1 {
		return m[0][0]
	}
	if n == 2 {
		return m[0][0]*m[1][1] - m[0][1]*m[1][0]
	}
	cp := copyMatrix(m)
	det := 1.0
	for col := 0; col < n; col++ {
		pivot := -1
		for row := col; row < n; row++ {
			if math.Abs(cp[row][col]) > 1e-10 {
				pivot = row
				break
			}
		}
		if pivot == -1 {
			return 0
		}
		if pivot != col {
			cp[col], cp[pivot] = cp[pivot], cp[col]
			det *= -1
		}
		det *= cp[col][col]
		for row := col + 1; row < n; row++ {
			factor := cp[row][col] / cp[col][col]
			for k := col; k < n; k++ {
				cp[row][k] -= factor * cp[col][k]
			}
		}
	}
	return det
}

func transpose(m [][]float64) [][]float64 {
	rows, cols := len(m), len(m[0])
	t := make([][]float64, cols)
	for i := range t {
		t[i] = make([]float64, rows)
		for j := range t[i] {
			t[i][j] = m[j][i]
		}
	}
	return t
}

func inverse(m [][]float64) ([][]float64, error) {
	n := len(m)
	aug := make([][]float64, n)
	for i := range aug {
		aug[i] = make([]float64, 2*n)
		copy(aug[i], m[i])
		aug[i][n+i] = 1
	}
	for col := 0; col < n; col++ {
		pivot := -1
		for row := col; row < n; row++ {
			if math.Abs(aug[row][col]) > 1e-10 {
				pivot = row
				break
			}
		}
		if pivot == -1 {
			return nil, fmt.Errorf("matrix is singular (not invertible)")
		}
		aug[col], aug[pivot] = aug[pivot], aug[col]
		scale := aug[col][col]
		for k := 0; k < 2*n; k++ {
			aug[col][k] /= scale
		}
		for row := 0; row < n; row++ {
			if row == col {
				continue
			}
			factor := aug[row][col]
			for k := 0; k < 2*n; k++ {
				aug[row][k] -= factor * aug[col][k]
			}
		}
	}
	result := make([][]float64, n)
	for i := range result {
		result[i] = aug[i][n:]
	}
	return result, nil
}

func multiply(a, b [][]float64) [][]float64 {
	rows, cols, inner := len(a), len(b[0]), len(b)
	c := make([][]float64, rows)
	for i := range c {
		c[i] = make([]float64, cols)
		for j := 0; j < cols; j++ {
			for k := 0; k < inner; k++ {
				c[i][j] += a[i][k] * b[k][j]
			}
		}
	}
	return c
}

func addMatrix(a, b [][]float64) [][]float64 {
	c := copyMatrix(a)
	for i := range c {
		for j := range c[i] {
			c[i][j] += b[i][j]
		}
	}
	return c
}

func scalarMultiply(m [][]float64, s float64) [][]float64 {
	c := copyMatrix(m)
	for i := range c {
		for j := range c[i] {
			c[i][j] *= s
		}
	}
	return c
}

func trace(m [][]float64) float64 {
	var t float64
	for i := range m {
		t += m[i][i]
	}
	return t
}

func rank(m [][]float64) int {
	cp := copyMatrix(m)
	rows, cols := len(cp), len(cp[0])
	r := 0
	for col := 0; col < cols && r < rows; col++ {
		pivot := -1
		for row := r; row < rows; row++ {
			if math.Abs(cp[row][col]) > 1e-10 {
				pivot = row
				break
			}
		}
		if pivot == -1 {
			continue
		}
		cp[r], cp[pivot] = cp[pivot], cp[r]
		for row := r + 1; row < rows; row++ {
			if math.Abs(cp[r][col]) < 1e-10 {
				continue
			}
			factor := cp[row][col] / cp[r][col]
			for k := col; k < cols; k++ {
				cp[row][k] -= factor * cp[r][k]
			}
		}
		r++
	}
	return r
}

func copyMatrix(m [][]float64) [][]float64 {
	c := make([][]float64, len(m))
	for i := range m {
		c[i] = make([]float64, len(m[i]))
		copy(c[i], m[i])
	}
	return c
}

func matrixToString(m [][]float64) string {
	var sb strings.Builder
	for _, row := range m {
		parts := make([]string, len(row))
		for i, v := range row {
			parts[i] = fmt.Sprintf("%8.4g", v)
		}
		sb.WriteString("[ " + strings.Join(parts, "  ") + " ]\n")
	}
	return sb.String()
}

func matrixToLatex(m [][]float64) string {
	var rows []string
	for _, row := range m {
		parts := make([]string, len(row))
		for i, v := range row {
			if v == math.Trunc(v) {
				parts[i] = fmt.Sprintf("%.0f", v)
			} else {
				parts[i] = fmt.Sprintf("%.4g", v)
			}
		}
		rows = append(rows, strings.Join(parts, " & "))
	}
	return `\left(\matrix{` + strings.Join(rows, ` \cr `) + `}\right)`
}

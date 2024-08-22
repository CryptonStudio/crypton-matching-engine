package list

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSimpleIteration(t *testing.T) {
	t.Run("empty list", func(t *testing.T) {
		l := NewList[int]()
		it := NewIterator(l)
		for it.Next() {
			t.Fatal("no cycle for empty list")
		}
	})

	t.Run("step iteration", func(t *testing.T) {
		l := NewList[int]()
		l.PushBack(1)
		l.PushBack(2)
		l.PushBack(3)
		it := NewIterator(l)
		require.True(t, it.Next())
		require.Equal(t, 1, it.Current().Value)
		require.True(t, it.Next())
		require.Equal(t, 2, it.Current().Value)
		require.True(t, it.Next())
		require.Equal(t, 3, it.Current().Value)
		require.False(t, it.Next())
	})

	t.Run("copy iteration", func(t *testing.T) {
		testCases := [][]int{
			{1},
			{1, 2, 3},
			{4, 3, 2, 1},
		}
		for _, tc := range testCases {
			l := NewList[int]()
			for _, v := range tc {
				l.PushBack(v)
			}
			it := NewIterator(l)
			result := []int{}
			for it.Next() {
				result = append(result, it.Current().Value)
			}
			require.Len(t, result, len(tc))
			for i := range tc {
				require.Equal(t, tc[i], result[i])
			}
		}
	})

	t.Run("consume iteration", func(t *testing.T) {
		testCases := [][]int{
			{1},
			{1, 2, 3},
			{4, 3, 2, 1},
		}
		for _, tc := range testCases {
			l := NewList[int]()
			for _, v := range tc {
				l.PushBack(v)
			}

			it := NewIterator(l)
			result := []int{}

			for it.Next() {
				result = append(result, it.Current().Value)
				_, err := l.Remove(it.Current())
				require.NoError(t, err)
			}

			require.Equal(t, 0, l.Len())
			require.Len(t, result, len(tc))

			for i := range tc {
				require.Equal(t, tc[i], result[i])
			}
		}
	})

	t.Run("odd consume iteration", func(t *testing.T) {
		source := []int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}
		expect := []int{1, 3, 5, 7, 9}
		l := NewList[int]()
		for _, v := range source {
			l.PushBack(v)
		}
		it := l.Iterator()
		result := []int{}
		visited := []int{}
		for it.Next() {
			visited = append(visited, it.Current().Value)
			if it.Current().Value%2 == 1 {
				result = append(result, it.Current().Value)
				_, err := l.Remove(it.Current())
				require.NoError(t, err)
			}
		}
		require.Equal(t, expect, result)
		require.Equal(t, source, visited)
	})

	t.Run("clean", func(t *testing.T) {
		l := NewList[int]()
		l.PushBack(1)
		l.PushBack(2)
		l.PushBack(3)
		it := l.Iterator()
		require.True(t, it.Next())
		require.Equal(t, 1, it.Current().Value)
		l.Clean()
		require.False(t, it.Next())
	})
}

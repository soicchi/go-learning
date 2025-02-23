# Overview

ゴルーチンやチャネルについて

# Goroutionの仕組み

```mermaid
flowchart

A["func main()"] --go Func1()--> B["func Func1()"]
A --go Func2()--> C["func Func2()"]
B --go Func3()--> D["func Func3()"]
B --go Func4()--> E["func Func4()"]
C --go Func5()--> F["func Func5()"]
C --go Func6()--> G["func Func6()"]
```

`main()`が実行されるとメインゴルーチンが生成される。

そこから`go`のprefixをつけて関数が呼ばれると新たにgoroutineを生成する。

`main()`の処理が終了するとすべてのgoroutineは強制的に終了する。

# channelの送信側と受信側の動作

channelにbufferが設定された場合とされてない場合の動作の違いについて説明する。

## channelにbufferが設定されていない場合

- 受信側の処理(<-ch)は送信側がchannelにデータを入れる(ch <- 1)までブロックされる
- 送信側の処理(ch <- 1)は受信側が値を受け取る(<-ch)までブロックされる

**sample code**
```go
func main() {
    ch := make(chan int)

    go func() {
        ch <- 1 // <-ch が実行されるまで処理は進まない
        fmt.Println("completed sending")
    }

    fmt.Println(<-ch) // ch <- 1 でchannelに値が入るまで処理は進まない
}
```

**output**
```sh
# fmt.Println("completed sending") の処理が重い処理ではないため下記の順序に出力される
completed sending
1
```

例えば、以下のように`ch <- 1`以降の処理を重くすると

```go
func main() {
    ch := make(chan int)

    // goroutine2が生成される
    go func() {
        ch <- 1 // <-ch が実行されるまで処理は進まない
        time.Sleep(5 * time.Second)
        fmt.Println("completed sending")
    }

    fmt.Println(<-ch) // ch <- 1 でchannelに値が入るまで処理は進まない
}
```

```sh
# 受信側の処理しか出力されない
1
```

これは`main()`が実行された際に生成されるメインのgoroutineが新たに生成したgoroutineの処理を待たずに処理を終了させてしまうため起こる。

```mermaid
flowchart

A[main関数実行] --> B[goroutine生成]
subgraph groutine 1
B --> C[main関数の処理]
C --> D[新たなgoroutine生成]
D --channelに1が入った--> G[chの中身の1を出力]
end
subgraph groutine 2
D --> E[channelに1を入れる]
E --> I[5秒待機]
I --処理される前にmain関数が終了--> F[❌completed sending]
end
F --❌--> H
G --> H[main関数の処理終了]
```

Go の main() は "メインゴルーチン" として動作し、main() の処理が終了すると、すべてのゴルーチンが強制終了する。

そのため、`sync.WaitGroup` を使用してゴルーチンの完了を待つことで、main() の終了を防ぐ。

```go
func main() {
	// with no buffer
	ch := make(chan int)
	var wg sync.WaitGroup
	wg.Add(1) // 処理を待つgoroutineの数をセット

    // goroutine2が生成される
	go func() {
		defer wg.Done() // wgのcounterを1減らす
		ch <- 1
		time.Sleep(5 * time.Second)
		fmt.Println("completed sending")
	}()

	fmt.Println(<-ch)

	wg.Wait() // wgのcounterが0になるまで待機
}
```

```mermaid
flowchart

A[main関数実行] --> B[goroutine生成]
subgraph groutine 1
B --> C[main関数の処理]
C --> D[新たなgoroutine生成]
D --channelに1が入った--> G[chの中身の1を出力]
G --> J[goroutine2が終わるまで待機]
end
subgraph groutine 2
D --> E[channelに1を入れる]
E --> I[5秒待機]
I --> F[completed sending]
end
F --> J
J --> H[main関数の処理終了]
```

## channelにbufferが設定されている場合

- 送信側(ch <- 1)の処理はchannelのbufferに空きがあれば、後続の処理に進む
- channelのbufferがいっぱいになった場合、bufferに空きができるまで後続処理を待機
- 受信側(<-ch)はchannelに値が入ってくるまで待機

```go
func main() {
	ch := make(chan int, 3)
	var wg sync.WaitGroup

	wg.Add(1)

	go func() {
		defer wg.Done()

		for i := 0; i < 3; i++ {
			ch <- 1 // bufferに空きがあれば即座に後続の処理に進む
			fmt.Println("sent")
			time.Sleep(1 * time.Second)
		}

		close(ch) // 受信側に値がもう入ってこないことを知らせる
	}()

	for v := range ch { // chがcloseされてない場合、ずっと待ち続けることになる。
		fmt.Println(v)
		fmt.Println("received")
	}

	wg.Wait()
}
```

# channelに対してfor-rangeループを利用する際の注意点

channelに対して`for-range`を利用する際はchannelを`close`するか、ループを`break` or `return`で抜けよう。

一般的なループと異なるchannelをfor-rangeループする場合、宣言できる引数は1つのみ

```go
// 一般的なfor-range
items := []int{1,2,3}

for i, num := range items { // i, numの二つの変数を宣言できる
    fmt.Println(i)
    fmt.Println(num)
}


// channelのfor-range
ch := make(chan int, 3)

// do something to add value to ch
close(ch) // channelをcloseしないと、以下のfor-rangeループが回り続けてしまう

for v := range ch { // v(channelに入っている値)のみ宣言可能
    fmt.Println(v)
}
```

channelを利用してfor-rangeループを行う場合、そのchannelが`close`されるかループが`break` or `return`されるまでループされる。

なので明示的に`close(ch)`や`break` or `return`を記載していないのでループが回り続けてしまう。

# デッドロック

複数のchannelを利用している場合、デットロックの可能性を考慮しなければならない。

```go
func main() {
	ch1 := make(chan int)
	ch2 := make(chan int)

    // goroutine2
	go func() {
		v := 1
		ch1 <- v // ch1から値が受け取られるまで待機
		v2 := <-ch2 // ch2に値が入るまで待機
		fmt.Println(v, v2)
	}()

	v := 2
	ch2 <- v // ch2から値が受け取られるまで待機
	v1 := <-ch1 // ch1に値が入るまで待機
	fmt.Println(v, v1)
}
```

main goroutineがch2から値が受け取られるまで`ch2 <- v`以降の処理をブロックし、

goroutine2ではch1から値が受け取られるまで`ch1 <- v`以降の処理をブロックする。

そのため二つのgoroutine間で処理がブロックされてしまいデッドロックになる。

```mermaid
flowchart

subgraph main goroutine
A[main関数実行] --> B[goroutine2生成]
B --> J[ch2にvを入れる]
J --> C[ch2から値が受け取られるまで待機]
C --❌deadlock--> E[ch2に値が入るまで待機]
E --❌--> L[ch1に値が入ってくるまで待機]
L --❌--> M[ch1から値を取得してv1に代入]
M --❌--> H[v, v1出力]
end
subgraph goroutine2
B --> K[ch1にvを入れる]
K --> D[ch1から値が受け取られるまで待機]
D --❌deadlock--> F[ch2に値が入るまで待機]
F --❌--> N[ch2から値を受け取りv2に代入]
N --❌--> G[v, v2出力]
end
H --❌--> I[main関数終了]
G --❌--> I
```

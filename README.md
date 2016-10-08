#Concurrent and Parallel Systems Assignment

For this assignment I was required to design, implement, test and analyse the performance of one or more concurrent / parallel algorithms. The Go programming language was used for implementation due to its built in support for concurrency.

##Problems being solved

In this assignment I implemented two algorithms to solve problems from two separate domains. One algorithm (image_process) performs simple image processing where an image kernel or convolution matrix is applied to pixels in an image to produce a new filtered image with some effect (blur, sharpen, edges). 

The second algorithm (caps_grep) is a very basic implementation of a “grep” program that counts the number of occurrences of a target word in all the text files in a given directory and its subdirectories recursively, it also notes the location of the word in each file with a line number and column number. 

##Core algorithm that solves each problem
At their core each implementation makes use of a key algorithm or piece of code that does the low level work to solve the problem. 

The image processing implementation uses a function called **applyKernelPixel**, it works by taking the kernel and calculates what a pixel in the source image, specified by x & y coordinates, will look like with the kernel applied to it. This is done by taking the values of the pixels surrounding the target pixel, multiplying them by their corresponding values in the kernel, then summing the results up to get a final value for the output pixel. This value is then put into the output image at the specified x & y location. 

The simple grep implementation makes use of a function called **searchBytes**. The function takes a byte buffer and a target string, it works by iterating over the byte buffer and counting the number of consecutive bytes that match the target string, when this reaches the length of the target string a match is recorded and the location of the match in the file is noted. The function returns the total number of matches and all the locations in the file where a match was recorded. 

##How the Sequential Version Works 
The sequential version of the simple grep implementation works by generating a list of all the files to open in the given directory and all directories below it, this list of files is then iterated over in a sequential fashion and searched using the **searchBytes** function. Each file is opened and the contents loaded into a byte buffer which is passed, with the target string, to the **searchBytes** function. 

The sequential version of the image processing implementation works by loading a source image from a file into a buffer and then iterating over every pixel in the buffer, using the **applyKernelPixel** function to calculate each pixel’s value with the kernel applied to it. The x & y coordinates of each pixel is passed into the function sequentially and the function then places the result into an output buffer to create the output image. 

##How the Parallel Version Works

Being written in Go both implementations make use of its concurrency features, by design these implementations are “concurrent” as multiple threads can make progress through the workload they are given even though they might not be running at the same time (multitasking via time­slicing) due to only being able to use one CPU. Parallelism is only achieved at run­time, if multiple threads can execute simultaneously, this can only be done with multiple CPUs. In Go the max amount of CPUs that the program can use can be specified.

###Simple Grep

The concurrent / parallel version of the simple grep implementation works by generating a list of all the files to open in the given directory and all directories below it, a “worker” Go routine is then launched for each file in the list, the routines then wait to be given “jobs” on the job channel. 

A job is the name of the file a given routine will search, before beginning searching the routine has to acquire the okay to begin from a channel that counts the amount of files that are open, this prevents too many files being opened. When it acquires the okay the counter goes up on the channel (which has a max, it will block routines from continuing ­ and thereby opening files, until the counter comes back down).

When the routine is finished searching it puts the results from the search into channels which are then picked up by “collector” routines, after this the routine releases the okay it acquired earlier from the “opened file counter” channel so other routines can continue. 

The search is complete when all the worker routines finish and all the collector routine have put the final result together. 

###Image Processing

The concurrent / parallel version of the image processing implementation works by loading a source image from a file into a buffer and then iterating over each row in the source image, each row is then sent to a Go routine to have the kernel applied to it. The Go routine iterates over the row of pixels it is given using the **applyKernelPixel** function on each pixel, since each pixel has its own destination in the output image buffer it can be written to by each Go routine without the use of channels to share access to the buffer.

The algorithm is complete when all the Goroutines have finished executing and the output image is then saved to disk.

See the [Report](https://github.com/kevinchar93/University_CAPS_Assignment/blob/master/CAPS_AssignmentReport_KevinCharles.pdf) for more details and a simple performance analysis.

## License

Copyright © 2016 Kevin Charles

Distributed under the MIT License

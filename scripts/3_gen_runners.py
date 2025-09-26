#!/usr/bin/python3

configs = ['normal', 'infinite-l2'] # 'l1-prefetcher']

benchmarks = [
    "aes",
    "atax",
    "bfs",
    "bicg",
    "bitonicsort",
    "conv2d",
    "fastwalshtransform",
    "fir",
    "floydwarshall",
    "kmeans",
    "matrixmultiplication",
    "matrixtranspose",
    "nbody",
    "pagerank",
    "relu",
    "simpleconvolution",
    "spmv",
    "stencil2d"
]

for config in configs:
    for benchmark in benchmarks:
        print(config, benchmark)
        submit_file_name = config + '/' + benchmark + ".sh"
        submit_file = open(submit_file_name, "w")
        submit_file.write("#!/bin/bash\n")
        submit_file.write(f"cd {benchmark}\n")
        # submit_file.write("cd " + benchmark + "\n")
        submit_file.write(f'echo {config} >> timing_report.txt\n')
        submit_file.write("{ time ")
        submit_file.write("./" + benchmark + " ")
        submit_file.write("-timing ")
        submit_file.write("-report-all ")

        if config == 'l1-prefetcher':
            submit_file.write("-use-l1-prefetcher=true ")
        
        if config == 'l2-prefetcher':
            submit_file.write("-use-l2-prefetcher=true ")
        
        if config == 'combined':
            submit_file.write("-use-l1-prefetcher=true -use-l2-prefetcher=true ")
        
        if config == "infinite-l2":
            submit_file.write("-use-inf-l2=true ")

        # Set benchmark specific parameters
        if benchmark == 'aes':
            submit_file.write("-length=33554432 ")  # Similar scale to other benchmarks -dram=21 
            # submit_file.write("-unified-gpus=1,2,3,4 ")
        elif benchmark == 'atax':
            submit_file.write("-x=12288 -y=12288 ")  #  Similar scale to other benchmarks -dram=45 
            # submit_file.write("-unified-gpus=1,2,3,4 ")
        elif benchmark == 'bfs':
            submit_file.write("-node=131072 ")  # GRIT -dram=22 
            # submit_file.write("-unified-gpus=1,2,3,4 ")
        elif benchmark == 'bicg':
            submit_file.write("-x=12288 -y=12288 ")  # Similar scale to other benchmarks -dram=45 
            # submit_file.write("-unified-gpus=1,2,3,4 ")
        elif benchmark == 'bitonicsort':
            submit_file.write("-length=6553600 ")   # GRIT -dram=21 
            # submit_file.write("-unified-gpus=1,2,3,4 ")
            # submit_file.write("-gpus=1,2,3,4 ")
        elif benchmark == 'concurrentkernel':
            submit_file.write(" ")  # Based on typical concurrent workload parameters
        elif benchmark == 'concurrentworkload':
            submit_file.write("-workload-size=8192 -num-workloads=32 ")  # Similar to other memory-intensive benchmarks
        elif benchmark == 'conv2d':                 # GRIT
            submit_file.write("-N=1 -C=1 -H=1024 -W=1024 -dram=64 ") # submit_file.write("-unified-gpus=1,2,3,4 ") submit_file.write("-N=1 -C=1 -H=128 -W=128  ") # 
        elif benchmark == 'fastwalshtransform':     # MCM GPU Code  &   Griffin
            submit_file.write("-length=8388608 ") # -dram=28 
            # submit_file.write("-unified-gpus=1,2,3,4 ")
        elif benchmark == 'fft':
            submit_file.write("-MB=8192 -passes=64 ")  # Similar to other transform benchmarks
        elif benchmark == 'fir':
            submit_file.write("-length=19824640 ") # submit_file.write("-length=300000 -magic-memory-copy ")   # GRIT -dram=108 
            # submit_file.write("-unified-gpus=1,2 ") # submit_file.write("-unified-gpus=1,2,3,4 ")
        elif benchmark == 'floydwarshall':
            submit_file.write("-node=2400 -iter=16 ")       # Griffin -dram=31
            # submit_file.write("-unified-gpus=1,2,3,4 ")
        elif benchmark == 'im2col':
            submit_file.write("-N=32 -C=3 -H=1024 -W=1024")  # Similar to conv2d but smaller
        elif benchmark == 'kmeans':                 # Griffin   -dram=36 
            submit_file.write("-points=262144 -features=32 -clusters=8 -max-iter=1 ")
            # submit_file.write("-unified-gpus=1,2,3,4 ")
        elif benchmark == 'lenet':
            submit_file.write("-epoch=10 ")
            submit_file.write("-max-batch-per-epoch=10 ")
            submit_file.write("-enable-testing=true ")
        elif benchmark == 'matrixmultiplication':
            submit_file.write("-x=2048 -y=2048 -z=1024 ") # submit_file.write("-x=1024 -y=1024 -z=512  ")   # GRIT      -dram=23 
            # submit_file.write("-unified-gpus=1,2 ") # submit_file.write("-unified-gpus=1,2,3,4 ")
            # submit_file.write("-gpus=1,2,3,4 ")
        elif benchmark == 'matrixtranspose':
            submit_file.write("-width=2400 ")       # Griffin   -dram=31 
            # submit_file.write("-unified-gpus=1,2,3,4 ")
            # submit_file.write("-gpus=1,2,3,4 ")
        elif benchmark == 'memcopy':
            submit_file.write("")  # Similar to fastwalshtransform
        elif benchmark == 'minerva':
            submit_file.write(
                "-epoch=10 -max-batch-per-epoch=5 -batch-size=128 -enable-testing=true -enable-verification=true ")
        elif benchmark == 'nbody':
            submit_file.write("-particles=1048576 -iter=45 ")  # Added iterations similar to other benchmarks   -dram=23 
            # submit_file.write("-unified-gpus=1,2,3,4 ")
        elif benchmark == 'nw':
            submit_file.write("-length=2048 ")
        elif benchmark == 'pagerank':           # Griffin   --dram=27 
            submit_file.write("-node=433107 -sparsity=0.001 -iterations=1 ")
            # submit_file.write("-unified-gpus=1,2,3,4 ")
        elif benchmark == 'relu':
            submit_file.write("-length=8388608 ")  # Added batch-size similar to neural network benchmarks -dram=23 
            # submit_file.write("-unified-gpus=1,2,3,4 ")
        elif benchmark == 'simpleconvolution':
            submit_file.write("-width=4096 -height=4096 ")  # GRIT  -dram=91 
            # submit_file.write("-unified-gpus=1,2,3,4 ")
        elif benchmark == 'spmv':
            submit_file.write("-dim=20400 -sparsity=0.01 ") # Similar scale to other benchmarks -dram=23
            # submit_file.write("-unified-gpus=1,2,3,4 ")
        elif benchmark == 'stencil2d':
            submit_file.write("-row=2048 -col=2048 -iter=10 ")  # GRIT  -dram=23 
            # submit_file.write("-unified-gpus=1,2,3,4 ")
        elif benchmark == 'vgg16':
            submit_file.write("-epoch=10 ")
            submit_file.write("-max-batch-per-epoch=20 ")
            submit_file.write("-batch-size=32 ")
            submit_file.write("-enable-testing=true ")
        elif benchmark == 'xor':
            submit_file.write("")  # Similar scale to other compute benchmarks

        submit_file.write(";} >>log.txt 2>> timing_report.txt")
        submit_file.close()  # Close the file after writing

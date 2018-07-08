import re
import subprocess
import numpy as np
import pandas as pd
import os
import argparse
import sys
sys.path.append(os.getcwd() + '/../common/')
from benchmarking import *

data_columns = ['benchmark', 'env', 'inst', 'count', 'time']

def generate_benchmark(inst, count):
    with open('microbench/kernels_template.asm', 'r') as template_file:
        template = template_file.read()

    with open('microbench/kernels.asm', 'w') as kernel_file:
        kernel_file.write(template)
        for i in range(0, count):
            kernel_file.write(inst + '\n')
        kernel_file.write('s_endpgm\n')

    p = subprocess.Popen('make kernels.hsaco', cwd='microbench',
                         shell=True)
    p.wait()


def run_on_simulator(inst):
    """ run benchmark and retuns a data frame that represents its result """
    data = pd.DataFrame(columns=data_columns)

    process = subprocess.Popen("go build", shell=True, cwd='.',
                            stdout=subprocess.DEVNULL)
    process.wait()

    for num_inst in range(0, 129, 1):
        print('On GPU: {0}, {1}'.format(inst, num_inst))
        generate_benchmark(inst, num_inst)
        duration = run_benchmark_on_simulator('./alu -timing', os.getcwd())
        data = data.append(
            pd.DataFrame([['alu', 'sim', inst, num_inst, duration]],
                            columns=data_columns),
            ignore_index=True,
        )

    return data


def run_on_gpu(inst, repeat):
    data = pd.DataFrame(columns=data_columns)

    for num_inst in range(0, 129, 1):
        print('On GPU: {0}, {1}'.format(inst, num_inst))
        generate_benchmark(inst, num_inst)
        for i in range(0, repeat):
            duration = run_benchmark_on_gpu(
                './kernel', os.getcwd() + '/microbench/')
            data = data.append(
                pd.DataFrame([['alu', 'gpu', inst, num_inst, duration]],
                             columns=data_columns),
                ignore_index=True,
            )

    return data

def parse_args():
    parser = argparse.ArgumentParser(description='ALU microbenchmark')
    parser.add_argument('--gpu', dest='gpu', action='store_true')
    parser.add_argument('--sim', dest='sim', action='store_true')
    parser.add_argument('--repeat', type=int, default=20)
    args = parser.parse_args()
    return args


def main():
    args = parse_args()

    insts = ['v_add_f32 v1, v2, v3']

    if args.gpu:
        for inst in insts:
            data = pd.DataFrame(columns=data_columns)
            results = run_on_gpu(inst, args.repeat)
            data = data.append(results, ignore_index=True)
            data.to_csv('gpu.csv')

    if args.sim:
        for inst in insts:
            data = pd.DataFrame(columns=data_columns)
            results = run_on_simulator(inst)
            data = data.append(results, ignore_index=True)
            data.to_csv('sim.csv')


if __name__ == '__main__':
    main()


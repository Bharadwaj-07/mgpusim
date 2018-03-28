__kernel void microbench(__global int* v1,  uint32 num_bits,__global int* output){

  uint tid = get_global_id(0);
  output[tid] = v1[tid] >> num_bits;
}

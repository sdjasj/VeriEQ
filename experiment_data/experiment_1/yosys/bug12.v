module top (in1, out2);
    input wire signed [21:18] in1;
    output wire  [24:5] out2;

    assign out2 = in1 >> 32;

endmodule
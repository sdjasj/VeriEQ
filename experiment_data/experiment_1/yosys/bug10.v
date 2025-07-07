5099module top (in1, out40);

    input wire  [3:0] in1;
    output wire [6:5] out40;

    assign out40 = (in1[1:1] >> 32'hffffffffffff);
endmodule
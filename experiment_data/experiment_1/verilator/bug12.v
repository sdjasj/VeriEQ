`timescale 1ns/1ps
`define stop $stop
`define checkd(gotv, expv) \
do if ((gotv) !== (expv)) begin \
  $write("%%Error: %s:%0d:  got=%0d exp=%0d\n", `__FILE__, `__LINE__, (gotv), (expv)); \
  `stop; \
end while(0);
module top (out20);

reg in0;
reg signed [7:0] in4;
wire [1:0] wire_0;
output wire out20;

assign wire_0 = in4[0:0] ? ({{7{in4[3:1]}}, 12'd201} & 2'h2) : (!(in0) >> 9'b1111);
assign out20 = wire_0[0:0];

initial begin
  #10;
  in4 = 16'b1111_1110;
  in0 = 7'b0;
  #10;
  `checkd(out20, '0);
  $write("*-* All Finished *-*\n");
  $finish;
end

endmodule
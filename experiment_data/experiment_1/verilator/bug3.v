`timescale 1ns/1ps
module gymhnulbvj (in5, clock_10, clock_12, out18);

input wire [23:22] in5;
wire [29:1] wire_4;
reg reg_35;
output wire out18;
input wire clock_10;
input wire clock_12;

assign wire_4 = ~(in5[22:22]);
assign out18 = reg_35 ? 0 : !(!(~((wire_4[6:5] | 8'hc6))));
always @(posedge clock_10 or posedge clock_12) begin
  if (clock_12) begin
    reg_35 <= 0;
  end else begin
    reg_35 <= wire_4;
  end
end
endmodule